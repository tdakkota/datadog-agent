// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package logs

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/DataDog/datadog-agent/pkg/autodiscovery"
	coreConfig "github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/metadata"
	"github.com/DataDog/datadog-agent/pkg/util/log"

	"github.com/DataDog/datadog-agent/pkg/logs/client/http"
	"github.com/DataDog/datadog-agent/pkg/logs/config"
	"github.com/DataDog/datadog-agent/pkg/logs/metrics"
	"github.com/DataDog/datadog-agent/pkg/logs/scheduler"
	"github.com/DataDog/datadog-agent/pkg/logs/service"
	"github.com/DataDog/datadog-agent/pkg/logs/status"
)

const (
	// key used to display a warning message on the agent status
	invalidProcessingRules = "invalid_global_processing_rules"
	invalidEndpoints       = "invalid_endpoints"
)

var (
	// isRunning indicates whether logs-agent is running or not
	isRunning int32
	// logs-agent
	agent *Agent
	// ErrMainCtxCanceled Logs context canceled
	ErrMainCtxCanceled = errors.New("Main context was canceled")
)

// Start starts logs-agent
// getAC is a func returning the prepared AutoConfig. It is nil until
// the AutoConfig is ready, please consider using BlockUntilAutoConfigRanOnce
// instead of directly using it.
func Start(ctx context.Context, getAC func() *autodiscovery.AutoConfig) error {
	if IsAgentRunning() {
		return nil
	}

	// setup the sources and the services
	sources := config.NewLogSources()
	services := service.NewServices()

	// setup the config scheduler
	scheduler.CreateScheduler(sources, services)

	// setup the server config
	httpConnectivity := config.HTTPConnectivityFailure
	if endpoints, err := config.BuildHTTPEndpoints(); err == nil {
		httpConnectivity = http.CheckConnectivity(endpoints.Main)
	}
	endpoints, err := config.BuildEndpoints(httpConnectivity)
	if err != nil {
		message := fmt.Sprintf("Invalid endpoints: %v", err)
		status.AddGlobalError(invalidEndpoints, message)

		e := errors.New(message)
		log.Error("Could not start logs-agent: ", e)
	}
	status.CurrentTransport = status.TransportTCP
	if endpoints.UseHTTP {
		status.CurrentTransport = status.TransportHTTP
	}

	// setup the status
	status.Init(&isRunning, endpoints, sources, metrics.LogsExpvars)

	// setup global processing rules
	processingRules, err := config.GlobalProcessingRules()
	if err != nil {
		message := fmt.Sprintf("Invalid processing rules: %v", err)
		status.AddGlobalError(invalidProcessingRules, message)
		e := errors.New(message)
		log.Error("Could not start logs-agent: ", e)

		return e
	}

	// setup and start the agent
	agent = NewAgent(sources, services, processingRules, endpoints)
	log.Info("Starting logs-agent...")

	metadataTO := coreConfig.Datadog.GetInt("logs_config.logs_meta_timeout")
	if metadataTO > 0 {
		if coreConfig.Datadog.GetString("app_key") != "" {
			// poll for a certain amount of time
			err := metadata.Ready(ctx, endpoints.Main.Host, metadataTO)
			if err != nil {
				log.Infof("There was an issue waiting for the metadata: %v", err)
				if errors.Is(err, ErrMainCtxCanceled) {
					return err
				}
			}
		} else {
			log.Info("Application key required to wait for host tags on backend, skipping wait.")
		}
	}

	agent.Start()
	atomic.StoreInt32(&isRunning, 1)
	log.Info("logs-agent started")

	// add SNMP traps source forwarding SNMP traps as logs if enabled.
	if source := config.SNMPTrapsSource(); source != nil {
		log.Debug("Adding SNMPTraps source to the Logs Agent")
		sources.AddSource(source)
	}

	// adds the source collecting logs from all containers if enabled,
	// but ensure that it is enabled after the AutoConfig initialization
	if source := config.ContainerCollectAllSource(); source != nil {
		go func() {
			BlockUntilAutoConfigRanOnce(getAC, time.Millisecond*time.Duration(coreConfig.Datadog.GetInt("ac_load_timeout")))
			log.Debug("Adding ContainerCollectAll source to the Logs Agent")
			sources.AddSource(source)
		}()
	}

	return nil
}

// BlockUntilAutoConfigRanOnce blocks until the AutoConfig has been run once.
// It also returns after the given timeout.
func BlockUntilAutoConfigRanOnce(getAC func() *autodiscovery.AutoConfig, timeout time.Duration) {
	now := time.Now()
	for {
		time.Sleep(100 * time.Millisecond) // don't hog the CPU
		if getAC().HasRunOnce() {
			return
		}
		if time.Since(now) > timeout {
			log.Error("BlockUntilAutoConfigRanOnce timeout after", timeout)
			return
		}
	}
}

// Stop stops properly the logs-agent to prevent data loss,
// it only returns when the whole pipeline is flushed.
func Stop() {
	log.Info("Stopping logs-agent")
	if IsAgentRunning() {
		if agent != nil {
			agent.Stop()
			agent = nil
		}
		if scheduler.GetScheduler() != nil {
			scheduler.GetScheduler().Stop()
		}
		status.Clear()
		atomic.StoreInt32(&isRunning, 0)
	}
	log.Info("logs-agent stopped")
}

// IsAgentRunning returns true if the logs-agent is running.
func IsAgentRunning() bool {
	return status.Get().IsRunning
}

// GetStatus returns logs-agent status
func GetStatus() status.Status {
	return status.Get()
}
