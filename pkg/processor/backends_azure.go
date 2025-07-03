//go:build azure_only

package processor

import (
	"github.com/liifi/secretinit/pkg/backend"
)

// RegisterAllBackends registers only git and azure backends
func RegisterAllBackends() map[string]func() (backend.Backend, error) {
	return map[string]func() (backend.Backend, error){
		"git":   func() (backend.Backend, error) { return &backend.GitBackend{}, nil },
		"azure": func() (backend.Backend, error) { return backend.NewAzureBackend() },
	}
}
