package api

var telemetryProxyEndpointsConfig = proxyEndpointsConfig{
	name:                      "telemetry",
	enabledConfig:             "apm_config.telemetry.enabled",
	ddURLConfig:               "apm_config.telemetry.dd_url",
	additionalEndpointsConfig: "apm_config.telemetry.additional_endpoints",
	urlTemplate:               "https://instrumentation-telemetry-intake.%s",
	defaultURL:                "https://instrumentation-telemetry-intake.datadoghq.com/",
	additionalDirector:        nil,
}
