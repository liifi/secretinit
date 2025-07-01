package backend

// Backend defines the interface for retrieving secrets from a specific backend.
type Backend interface {
	RetrieveSecret(service, resource, keyPath string) (string, error)
}
