package stats

import (
	"github.com/DataDog/datadog-agent/pkg/trace/watchdog"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/sketches-go/ddsketch/pb/sketchpb"
	"github.com/golang/protobuf/proto"
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/info"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/sketches-go/ddsketch"
)

const (
	pipelineBucketDuration       = 10 * time.Second
)

type pipelineStatsPoint struct {
	service string
	receivingPipelineName string
	summary *ddsketch.DDSketch
}

type pipelineAggregationKey struct {
	env string
	hostname string
	version string
}

type pipelineBucket struct {
	pipelineStats map[pipelineAggregationKey]map[uint64]pipelineStatsPoint
}

func (b *pipelineBucket) add(bucket pb.ClientPipelineStatsBucket, env, hostname, version string) {
	key := pipelineAggregationKey{
		env: env,
		hostname: hostname,
		version: version,
	}
	points, ok := b.pipelineStats[key]
	if !ok {
		points = make(map[uint64]pipelineStatsPoint)
	}
	for _, p := range bucket.Stats {
		var pbSummary *sketchpb.DDSketch
		err := proto.Unmarshal(p.Summary, pbSummary)
		if err != nil {
			log.Error("error decoding sketch: %v.", err)
			continue
		}
		summary, err := ddsketch.FromProto(pbSummary)
		if err != nil {
			log.Error("error building ddsketch from proto: %v.", err)
			continue
		}
		if point, ok := points[p.PipelineHash]; ok {
			// todo[piochelepiotr] Add check
			err := point.summary.MergeWith(summary)
			if err != nil {
				log.Error("error merging sketches: %v.", err)
				continue
			}
			continue
		}
		points[p.PipelineHash] = pipelineStatsPoint{
			receivingPipelineName: p.ReceivingPipelineName,
			service: p.Service,
			summary: summary,
		}
	}
}

func (b *pipelineBucket) export(start uint64, duration uint64) (p []pb.ClientPipelineStatsPayload) {
	for key, bucket := range b.pipelineStats {
		clientBucket := pb.ClientPipelineStatsBucket{
			Start: start,
			Duration: duration,
		}
		for hash, point := range bucket {
			summary, err := proto.Marshal(point.summary.ToProto())
			if err != nil {
				log.Error("error serializing ddsketch: %v", err)
				continue
			}
			clientBucket.Stats = append(clientBucket.Stats, pb.ClientGroupedPipelineStats{
				PipelineHash: hash,
				Service: point.service,
				ReceivingPipelineName: point.receivingPipelineName,
				Summary: summary,
			})
		}
		if len(clientBucket.Stats) > 0 {
			p = append(p, pb.ClientPipelineStatsPayload{
				Env: key.env,
				Version: key.version,
				Hostname: key.hostname,
				Stats: []pb.ClientPipelineStatsBucket{clientBucket},
			})
		}
	}
	return p
}

// PipelineStatsAggregator aggregates pipeline stats
type PipelineStatsAggregator struct {
	In      chan pb.ClientPipelineStatsPayload
	out     chan pb.PipelineStatsPayload
	buckets map[int64]*pipelineBucket

	flushTicker   *time.Ticker
	agentEnv      string
	agentHostname string

	exit chan struct{}
	done chan struct{}
}

// NewPipelineStatsAggregator initializes a new aggregator.
func NewPipelineStatsAggregator(conf *config.AgentConfig, out chan pb.PipelineStatsPayload) *PipelineStatsAggregator {
	return &PipelineStatsAggregator{
		flushTicker:   time.NewTicker(time.Second),
		In:            make(chan pb.ClientPipelineStatsPayload, 10),
		// todo[piochelepiotr] Should we group multiple buckets from the same tracer into the same flushed payload?
		buckets:       make(map[int64]*pipelineBucket, 20),
		out:           out,
		agentEnv:      conf.DefaultEnv,
		agentHostname: conf.Hostname,
		exit:          make(chan struct{}),
		done:          make(chan struct{}),
	}
}

// Start starts the aggregator.
func (a *PipelineStatsAggregator) Start() {
	go func() {
		defer watchdog.LogOnPanic()
		for {
			select {
			case t := <-a.flushTicker.C:
				a.flushOnTime(t)
			case input := <-a.In:
				a.add(input)
			case <-a.exit:
				a.flushAll()
				close(a.done)
				return
			}
		}
	}()
}

// Stop stops the aggregator. Calling Stop twice will panic.
func (a *PipelineStatsAggregator) Stop() {
	close(a.exit)
	<-a.done
}

// flushOnTime flushes all buckets up to flushTs, except the last one.
func (a *PipelineStatsAggregator) flushOnTime(now time.Time) {
	ts := now.UnixNano()
	duration := bucketDuration.Nanoseconds()
	for start, b := range a.buckets {
		if ts > start + duration {
			a.flush(b.export(uint64(start), uint64(duration)))
			delete(a.buckets, start)
		}
	}
}

func (a *PipelineStatsAggregator) flushAll() {
	for start, b := range a.buckets {
		a.flush(b.export(uint64(start), uint64(pipelineBucketDuration.Nanoseconds())))
	}
}

func (a *PipelineStatsAggregator) add(p pb.ClientPipelineStatsPayload) {
	for _, clientBucket := range p.Stats {
		clientBucketStart := time.Unix(0, int64(clientBucket.Start)).Truncate(pipelineBucketDuration)
		ts := clientBucketStart.Unix()
		b, ok := a.buckets[ts]
		if !ok {
			b = &pipelineBucket{pipelineStats: make(map[pipelineAggregationKey]map[uint64]pipelineStatsPoint)}
			a.buckets[ts] = b
		}
		b.add(clientBucket, p.Env, p.Hostname, p.Version)
	}
}

func (a *PipelineStatsAggregator) flush(p []pb.ClientPipelineStatsPayload) {
	if len(p) == 0 {
		return
	}
	a.out <- pb.PipelineStatsPayload{
		Stats:          p,
		AgentEnv:       a.agentEnv,
		AgentHostname:  a.agentHostname,
		AgentVersion:   info.Version,
	}
}
