package backend

import (
	"testing"
)

func TestAWSBackend_extractJSONKey(t *testing.T) {
	tests := []struct {
		name        string
		secretValue string
		keyPath     string
		want        string
		wantErr     bool
	}{
		{
			name:        "simple key extraction",
			secretValue: `{"username": "testuser", "password": "testpass"}`,
			keyPath:     "username",
			want:        "testuser",
			wantErr:     false,
		},
		{
			name:        "nested key extraction",
			secretValue: `{"database": {"username": "dbuser", "password": "dbpass"}}`,
			keyPath:     "database.username",
			want:        "dbuser",
			wantErr:     false,
		},
		{
			name:        "non-string value",
			secretValue: `{"port": 5432, "enabled": true}`,
			keyPath:     "port",
			want:        "5432",
			wantErr:     false,
		},
		{
			name:        "boolean value",
			secretValue: `{"port": 5432, "enabled": true}`,
			keyPath:     "enabled",
			want:        "true",
			wantErr:     false,
		},
		{
			name:        "missing key",
			secretValue: `{"username": "testuser"}`,
			keyPath:     "password",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "invalid JSON",
			secretValue: `not json`,
			keyPath:     "username",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "null value",
			secretValue: `{"username": null}`,
			keyPath:     "username",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "deep nested path",
			secretValue: `{"app": {"db": {"primary": {"user": "admin", "pass": "secret123"}}}}`,
			keyPath:     "app.db.primary.pass",
			want:        "secret123",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractJSONKey(tt.secretValue, tt.keyPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractJSONKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractJSONKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewAWSBackend(t *testing.T) {
	// This test will only pass if AWS credentials are configured
	// It's mainly for ensuring the constructor doesn't panic
	backend, err := NewAWSBackend()
	if err != nil {
		t.Logf("NewAWSBackend() failed (expected if AWS credentials not configured): %v", err)
		return
	}
	if backend == nil {
		t.Error("NewAWSBackend() returned nil backend")
	} else {
		if backend.secretsClient == nil {
			t.Error("NewAWSBackend() returned backend with nil secretsClient")
		}
		if backend.ssmClient == nil {
			t.Error("NewAWSBackend() returned backend with nil ssmClient")
		}
	}
}
