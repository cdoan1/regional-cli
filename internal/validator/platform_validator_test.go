package validator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestAWSConfig creates a test AWS config with static credentials
func createTestAWSConfig() aws.Config {
	return aws.Config{
		Region: "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider(
			"AKIAIOSFODNN7EXAMPLE",
			"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"",
		),
	}
}

func TestPlatformValidator_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify endpoint
		assert.Equal(t, "/prod/v0/live", r.URL.Path)

		// Verify AWS SigV4 headers are present
		assert.NotEmpty(t, r.Header.Get("Authorization"))
		assert.NotEmpty(t, r.Header.Get("X-Amz-Date"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	awsConfig := createTestAWSConfig()
	validator := NewPlatformValidator(server.URL, awsConfig)
	result, err := validator.Validate(context.Background())

	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Contains(t, result.APIVersion, "status")
	assert.Empty(t, result.ErrorMessage)
}

func TestPlatformValidator_NoURL(t *testing.T) {
	awsConfig := createTestAWSConfig()
	validator := NewPlatformValidator("", awsConfig)
	result, err := validator.Validate(context.Background())

	assert.Error(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessage, "Platform API URL is not configured")
}

func TestPlatformValidator_APIDown(t *testing.T) {
	awsConfig := createTestAWSConfig()
	validator := NewPlatformValidator("http://localhost:99999", awsConfig)
	result, err := validator.Validate(context.Background())

	assert.Error(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessage, "Failed to connect")
}

func TestPlatformValidator_BadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer server.Close()

	awsConfig := createTestAWSConfig()
	validator := NewPlatformValidator(server.URL, awsConfig)
	result, err := validator.Validate(context.Background())

	assert.Error(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessage, "returned status")
}

func TestPlatformValidator_CorrectEndpoint(t *testing.T) {
	// Verify the validator uses /prod/v0/live endpoint
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	awsConfig := createTestAWSConfig()
	validator := NewPlatformValidator(server.URL, awsConfig)
	_, err := validator.Validate(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "/prod/v0/live", requestedPath)
}

func TestPlatformValidator_SigV4Headers(t *testing.T) {
	// Verify AWS SigV4 headers are added to the request
	var authHeader, dateHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		dateHeader = r.Header.Get("X-Amz-Date")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	awsConfig := createTestAWSConfig()
	validator := NewPlatformValidator(server.URL, awsConfig)
	_, err := validator.Validate(context.Background())

	require.NoError(t, err)
	assert.NotEmpty(t, authHeader, "Authorization header should be present")
	assert.Contains(t, authHeader, "AWS4-HMAC-SHA256", "Authorization should use SigV4")
	assert.NotEmpty(t, dateHeader, "X-Amz-Date header should be present")
}
