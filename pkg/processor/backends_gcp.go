//go:build gcp_only

package processor

import (
	"github.com/liifi/secretinit/pkg/backend"
)

// RegisterAllBackends registers only git and gcp backends
func RegisterAllBackends() map[string]func() (backend.Backend, error) {
	return map[string]func() (backend.Backend, error){
		"git": func() (backend.Backend, error) { return &backend.GitBackend{}, nil },
		"gcp": func() (backend.Backend, error) { return backend.NewGCPBackend() },
	}
}
