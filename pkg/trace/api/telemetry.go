package api

var telemetryProxyEndpointsConfig = proxyEndpointsConfig{
	name:                      "telemetry",
	enabledConfig:             "apm_config.telemetry.enabled",
	ddURLConfig:               "apm_config.telemetry.dd_url",
	additionalEndpointsConfig: "apm_config.telemetry.additional_endpoints",
	urlTemplate:               "https://intake.apm-telemetry.%s",
	defaultURL:                "https://all-http-intake.logs.datad0g.com/",
}
