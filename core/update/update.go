package update

import (
	"runtime"

	"portside/core/porthome"
)

type CheckResult struct {
	CurrentVersion string                    `json:"current_version"`
	Channel        string                    `json:"channel"`
	Source         string                    `json:"source"`
	Platform       string                    `json:"platform"`
	Endpoints      []porthome.UpdateEndpoint `json:"endpoints"`
	Status         string                    `json:"status"`
	Message        string                    `json:"message"`
}

func Check(version string) CheckResult {
	channel := "stable"
	source := "auto"
	endpoints := porthome.DefaultConfig("").Update.Endpoints
	if config, err := porthome.LoadConfig(); err == nil {
		channel = config.Update.Channel
		source = config.Update.Source
		endpoints = config.Update.Endpoints
	}

	return CheckResult{
		CurrentVersion: version,
		Channel:        channel,
		Source:         source,
		Platform:       runtime.GOOS + "/" + runtime.GOARCH,
		Endpoints:      endpoints,
		Status:         "not_checked",
		Message:        "network update checks are not implemented in this build",
	}
}
