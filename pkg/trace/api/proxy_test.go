package api

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	traceconfig "github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/stretchr/testify/assert"
)

var testingProxyEndpointsConfig = proxyEndpointsConfig{
	name:                      "testing",
	enabledConfig:             "apm_config.telemetry.enabled",
	ddURLConfig:               "apm_config.telemetry.dd_url",
	additionalEndpointsConfig: "apm_config.telemetry.additional_endpoints",
	urlTemplate:               "https://intake.testing.%s/",
	defaultURL:                "https://intake.testing.datadoghq.com/",
}

func TestGenericProxy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		slurp, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if body := string(slurp); body != "body" {
			t.Fatalf("invalid request body: %q", body)
		}
		if v := req.Header.Get("DD-API-KEY"); v != "123" {
			t.Fatalf("got invalid API key: %q", v)
		}
		if v := req.Header.Get("DD-Agent-Hostname"); v != "testing_hostname" {
			t.Fatalf("got invalid Hostname: %q", v)
		}
		if v := req.Header.Get("DD-Agent-Env"); v != "testing_env" {
			t.Fatalf("got invalid Agent env: %q", v)
		}

		_, err = w.Write([]byte("OK"))
		if err != nil {
			t.Fatal(err)
		}
	}))
	mockConfig("apm_config.telemetry.dd_url", srv.URL)
	req, err := http.NewRequest("POST", "dummy.com/path", strings.NewReader("body"))
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	c := &traceconfig.AgentConfig{}

	transport, err := testingProxyEndpointsConfig.proxyRoundTripper(c.NewHTTPTransport(), "123")
	testingProxyEndpointsConfig.newForwardingProxy(transport, "testing_hostname", "testing_env").ServeHTTP(rec, req)
	slurp, err := ioutil.ReadAll(rec.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(slurp) != "OK" {
		t.Fatal("did not proxy")
	}
}

func TestGenericProxyEndpoints(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		defer mockConfig("apm_config.telemetry.dd_url", "https://intake.testing.datadoghq.fr/")()
		urls, keys, err := testingProxyEndpointsConfig.proxyEndpoints("test_api_key")
		assert.NoError(t, err)
		assert.Equal(t, urls, makeURLs(t, "https://intake.testing.datadoghq.fr/"))
		assert.Equal(t, keys, []string{"test_api_key"})
	})
	t.Run("site", func(t *testing.T) {
		defer mockConfig("site", "datadoghq.eu")()
		urls, keys, err := testingProxyEndpointsConfig.proxyEndpoints("test_api_key")
		assert.NoError(t, err)
		assert.Equal(t, urls, makeURLs(t, "https://intake.testing.datadoghq.eu/"))
		assert.Equal(t, keys, []string{"test_api_key"})
	})

	t.Run("default", func(t *testing.T) {
		urls, keys, err := testingProxyEndpointsConfig.proxyEndpoints("test_api_key")
		assert.NoError(t, err)
		assert.Equal(t, urls, makeURLs(t, "https://intake.testing.datadoghq.com/"))
		assert.Equal(t, keys, []string{"test_api_key"})
	})

	t.Run("multiple", func(t *testing.T) {
		defer mockConfigMap(map[string]interface{}{
			"apm_config.telemetry.dd_url": "https://intake.testing.datadoghq.jp",
			"apm_config.telemetry.additional_endpoints": map[string][]string{
				"https://ddstaging.datadoghq.com": {"api_key_1", "api_key_2"},
				"https://dd.datad0g.com":          {"api_key_3"},
			},
		})()
		urls, keys, err := testingProxyEndpointsConfig.proxyEndpoints("api_key_0")
		assert.NoError(t, err)
		expectedURLs := makeURLs(t,
			"https://intake.testing.datadoghq.jp",
			"https://ddstaging.datadoghq.com",
			"https://ddstaging.datadoghq.com",
			"https://dd.datad0g.com",
		)
		expectedKeys := []string{"api_key_0", "api_key_1", "api_key_2", "api_key_3"}

		// Because we're using a map to mock the config we can't assert on the
		// order of the endpoints. We check the main endpoints separately.
		assert.Equal(t, urls[0], expectedURLs[0], "The main endpoint should be the first in the slice")
		assert.Equal(t, keys[0], expectedKeys[0], "The main api key should be the first in the slice")

		assert.ElementsMatch(t, urls, expectedURLs, "All urls from the config should be returned")
		assert.ElementsMatch(t, keys, keys, "All keys from the config should be returned")

		// check that we have the correct pairing between urls and api keys
		for i := range keys {
			for j := range expectedKeys {
				if keys[i] == expectedKeys[j] {
					assert.Equal(t, urls[i], expectedURLs[j])
				}
			}
		}
	})
}
