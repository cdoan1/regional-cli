package validator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlatformValidator_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","version":"1.0.0"}`))
	}))
	defer server.Close()

	validator := NewPlatformValidator(server.URL)
	result, err := validator.Validate(context.Background())

	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.ErrorMessage)
}

func TestPlatformValidator_NoURL(t *testing.T) {
	validator := NewPlatformValidator("")
	result, err := validator.Validate(context.Background())

	assert.Error(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessage, "Platform API URL is not configured")
}

func TestPlatformValidator_APIDown(t *testing.T) {
	validator := NewPlatformValidator("http://localhost:99999")
	result, err := validator.Validate(context.Background())

	assert.Error(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessage, "Failed to connect")
}

func TestPlatformValidator_BadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	validator := NewPlatformValidator(server.URL)
	result, err := validator.Validate(context.Background())

	assert.Error(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessage, "Platform API returned status")
}
