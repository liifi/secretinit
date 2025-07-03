//go:build !aws_only && !gcp_only && !azure_only && !git_only

package processor

import (
	"github.com/liifi/secretinit/pkg/backend"
)

// RegisterAllBackends registers all available backends
func RegisterAllBackends() map[string]func() (backend.Backend, error) {
	return map[string]func() (backend.Backend, error){
		"git":   func() (backend.Backend, error) { return &backend.GitBackend{}, nil },
		"aws":   func() (backend.Backend, error) { return backend.NewAWSBackend() },
		"gcp":   func() (backend.Backend, error) { return backend.NewGCPBackend() },
		"azure": func() (backend.Backend, error) { return backend.NewAzureBackend() },
	}
}
