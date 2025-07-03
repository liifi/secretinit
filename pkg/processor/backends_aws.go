//go:build aws_only

package processor

import (
	"github.com/liifi/secretinit/pkg/backend"
)

// RegisterAllBackends registers only git and aws backends
func RegisterAllBackends() map[string]func() (backend.Backend, error) {
	return map[string]func() (backend.Backend, error){
		"git": func() (backend.Backend, error) { return &backend.GitBackend{}, nil },
		"aws": func() (backend.Backend, error) { return backend.NewAWSBackend() },
	}
}
