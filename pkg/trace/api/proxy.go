package api

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	stdlog "log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/trace/info"
	"github.com/DataDog/datadog-agent/pkg/trace/logutil"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

type proxyEndpointsConfig struct {
	name                      string
	enabledConfig             string // config entry for enabling disabling proxy endpoint, can be empty
	ddURLConfig               string // config entry for intake url *_DD_URL
	additionalEndpointsConfig string // config entry for adding additional, can be empty
	urlTemplate               string // template to build intake URL based on SITE config
	defaultURL                string
}

func (pec *proxyEndpointsConfig) proxyRoundTripper(client http.RoundTripper, apiKey string) (http.RoundTripper, error) {
	urls, apiKeys, err := pec.proxyEndpoints(apiKey)
	return &proxyMultiTransport{
		rt:      client,
		targets: urls,
		keys:    apiKeys,
		baseTag: "type:" + pec.name,
	}, err

}
func (pec *proxyEndpointsConfig) proxyEndpoints(apiKey string) (urls []*url.URL, apiKeys []string, err error) {
	main := pec.defaultURL
	if v := config.Datadog.GetString(pec.ddURLConfig); v != "" {
		main = v
	} else if site := config.Datadog.GetString("site"); site != "" {
		main = fmt.Sprintf(pec.urlTemplate, site)
	}
	u, err := url.Parse(main)
	if err != nil {
		// if the main intake URL is invalid we don't use additional endpoints
		return nil, nil, fmt.Errorf("error parsing main %s intake URL %s: %v", pec.name, main, err)
	}
	urls = append(urls, u)
	apiKeys = append(apiKeys, apiKey)

	if pec.additionalEndpointsConfig != "" && config.Datadog.IsSet(pec.additionalEndpointsConfig) {
		extra := config.Datadog.GetStringMapStringSlice(pec.additionalEndpointsConfig)
		for endpoint, keys := range extra {
			u, err := url.Parse(endpoint)
			if err != nil {
				log.Errorf("Error parsing additional %s intake URL %s: %v", pec.name, endpoint, err)
				continue
			}
			for _, key := range keys {
				urls = append(urls, u)
				apiKeys = append(apiKeys, key)
			}
		}
	}
	return urls, apiKeys, nil
}

// forwardingProxyHandler returns a new HTTP handler which will proxy requests to the profiling intakes.
// If the main intake URL can not be computed because of config, the returned handler will always
// return http.StatusInternalServerError along with a clarification.
func (r *HTTPReceiver) forwardingProxyHandler(pec *proxyEndpointsConfig) http.Handler {
	if pec.enabledConfig != "" && !config.Datadog.GetBool(pec.enabledConfig) {
		return pec.proxyDisabledHandler(nil)
	}
	transport, err := pec.proxyRoundTripper(r.conf.NewHTTPTransport(), r.conf.APIKey())
	if err != nil {
		return pec.proxyDisabledHandler(err)
	}

	return pec.newForwardingProxy(transport, r.conf.Hostname, r.conf.DefaultEnv)
}

func (pec *proxyEndpointsConfig) proxyDisabledHandler(err error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err != nil {
			msg := fmt.Sprintf("%s proxy forwarder is OFF: %v", pec.name, err)
			http.Error(w, msg, http.StatusInternalServerError)
		} else {
			msg := fmt.Sprintf("%s proxy forwarder is OFF", pec.name)
			http.Error(w, msg, http.StatusMethodNotAllowed)
		}
	})
}

// multiTransport sends HTTP requests to multiple targets using an
// underlying http.RoundTripper. API keys are set separately for each target.
// The target hostname
// When multiple endpoints are in use the response from the main endpoint
// is proxied back to the client, while for all aditional endpoints the
// response is discarded. There is no de-duplication done between endpoint
// hosts or api keys.
type proxyMultiTransport struct {
	rt      http.RoundTripper
	targets []*url.URL
	keys    []string
	baseTag string
}

func (m *proxyMultiTransport) overrideTarget(r *http.Request, targetUrl *url.URL, apiKey string) error {
	newTargetUrl := *targetUrl
	newPath := r.URL.Path
	if idx := strings.Index(newPath, "/"); idx != -1 {
		newPath = newPath[idx:]
	}

	r.Host = targetUrl.Host
	r.URL = &newTargetUrl
	r.Header.Set("DD-API-KEY", apiKey)
	return nil
}

// newForwardingProxy creates an http.ReverseProxy which can forward requests to
// one or more endpoints.
//
// The tags will be added as a header to all proxied requests.
// For more details please see multiTransport.
func (pec *proxyEndpointsConfig) newForwardingProxy(transport http.RoundTripper, agentHostname string, agentEnv string) http.Handler {
	director := func(req *http.Request) {
		req.Header.Set("Via", fmt.Sprintf("trace-agent %s", info.Version))
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to the default value
			// that net/http gives it: Go-http-client/1.1
			// See https://codereview.appspot.com/7532043
			req.Header.Set("User-Agent", "")
		}

		req.Header.Set("DD-Agent-Hostname", agentHostname)
		req.Header.Set("DD-Agent-Env", agentEnv)
	}
	logger := logutil.NewThrottled(5, 10*time.Second) // limit to 5 messages every 10 seconds
	return &httputil.ReverseProxy{
		Director:  director,
		ErrorLog:  stdlog.New(logger, fmt.Sprintf("%s.Proxy: ", pec.name), 0),
		Transport: transport,
	}
}

func (m *proxyMultiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	metrics.Count("datadog.trace_agent.proxy.request_count", 1, []string{m.baseTag}, 1)
	now := time.Now()
	defer func() {
		metrics.Timing("datadog.trace_agent.proxy.request_duration", time.Since(now), []string{m.baseTag}, 1)
	}()

	if len(m.targets) == 1 {
		m.overrideTarget(req, m.targets[0], m.keys[0])
		return m.rt.RoundTrip(req)
	}
	slurp, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	var (
		rresp *http.Response
		rerr  error
	)
	for i, u := range m.targets {
		newreq := req.Clone(req.Context())
		newreq.Body = ioutil.NopCloser(bytes.NewReader(slurp))
		m.overrideTarget(newreq, u, m.keys[i])
		if i == 0 {
			// given the way we construct the list of targets the main endpoint
			// will be the first one called, we return its response and error
			rresp, rerr = m.rt.RoundTrip(newreq)
			continue
		}

		if resp, err := m.rt.RoundTrip(newreq); err == nil {
			// we discard responses for all subsequent requests
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		} else {
			log.Error(err)
		}
	}
	return rresp, rerr
}
