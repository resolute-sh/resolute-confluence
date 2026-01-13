// Package confluence provides Confluence integration activities for resolute workflows.
package confluence

import (
	"github.com/resolute-sh/resolute/core"
	"go.temporal.io/sdk/worker"
)

const (
	ProviderName    = "resolute-confluence"
	ProviderVersion = "1.0.0"
)

// Provider returns the Confluence provider for registration.
func Provider() core.Provider {
	return core.NewProvider(ProviderName, ProviderVersion).
		AddActivity("confluence.FetchPages", FetchPagesActivity).
		AddActivity("confluence.FetchPage", FetchPageActivity).
		AddActivity("confluence.SearchCQL", SearchCQLActivity)
}

// RegisterActivities registers all Confluence activities with a Temporal worker.
func RegisterActivities(w worker.Worker) {
	core.RegisterProviderActivities(w, Provider())
}
