package adapters

import (
	"decentralized.host/pkg/providers"
)

// NewRegistry creates and initializes a registry with all built-in adapters
func NewRegistry() *providers.AdapterRegistry {
	registry := providers.NewAdapterRegistry()

	// Register all built-in adapters
	registry.Register("github", NewGitHubAdapter())
	registry.Register("vercel", NewVercelAdapter())
	registry.Register("supabase", NewSupabaseAdapter())
	registry.Register("docker", NewDockerAdapter())
	registry.Register("kubernetes", NewKubernetesAdapter())

	return registry
}
