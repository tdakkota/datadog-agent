package api

var telemetryProxyEndpointsConfig = proxyEndpointsConfig{
	name:                      "telemetry",
	mainConfig:                "apm_config.apm_dd_url",
	additionalEndpointsConfig: "apm_config.additional_endpoints",
	urlTemplate:               "https://intake.apm-telemetry.%s",
	defaultURL:                "https://all-http-intake.logs.datad0g.com/",
}
