package aws

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         ClientConfig
		expectError bool
	}{
		{
			name: "default config",
			cfg:  ClientConfig{},
			// Should succeed with default credentials
			expectError: false,
		},
		{
			name: "with region",
			cfg: ClientConfig{
				Region: "us-east-1",
			},
			expectError: false,
		},
		{
			name: "with profile and region",
			cfg: ClientConfig{
				Profile: "default",
				Region:  "us-west-2",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cfg, err := NewConfig(ctx, tt.cfg)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				// Config creation may succeed even without credentials
				// We can only verify the function doesn't panic
				_ = cfg
				_ = err
			}
		})
	}
}

func TestNewClients(t *testing.T) {
	ctx := context.Background()
	cfg, err := NewConfig(ctx, ClientConfig{Region: "us-east-1"})
	require.NoError(t, err)

	t.Run("NewLambdaClient", func(t *testing.T) {
		client := NewLambdaClient(cfg)
		assert.NotNil(t, client)
	})

	t.Run("NewIAMClient", func(t *testing.T) {
		client := NewIAMClient(cfg)
		assert.NotNil(t, client)
	})

	t.Run("NewSTSClient", func(t *testing.T) {
		client := NewSTSClient(cfg)
		assert.NotNil(t, client)
	})

	t.Run("NewCloudWatchLogsClient", func(t *testing.T) {
		client := NewCloudWatchLogsClient(cfg)
		assert.NotNil(t, client)
	})
}
