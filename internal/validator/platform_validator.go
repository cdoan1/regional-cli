package validator

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// PlatformValidator validates Platform API connectivity
type PlatformValidator struct {
	apiURL     string
	httpClient *http.Client
}

// NewPlatformValidator creates a new Platform API validator
func NewPlatformValidator(apiURL string) *PlatformValidator {
	return &PlatformValidator{
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// PlatformValidationResult holds the result of Platform API validation
type PlatformValidationResult struct {
	Valid        bool
	APIVersion   string
	ErrorMessage string
}

// Validate validates Platform API connectivity
func (v *PlatformValidator) Validate(ctx context.Context) (*PlatformValidationResult, error) {
	if v.apiURL == "" {
		return &PlatformValidationResult{
			Valid:        false,
			ErrorMessage: "Platform API URL is not configured",
		}, fmt.Errorf("API URL not configured")
	}

	// Create health check request
	req, err := http.NewRequestWithContext(ctx, "GET", v.apiURL+"/health", nil)
	if err != nil {
		return &PlatformValidationResult{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("Failed to create request: %v", err),
		}, err
	}

	// Execute request
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return &PlatformValidationResult{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("Failed to connect to Platform API: %v", err),
		}, err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return &PlatformValidationResult{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("Platform API returned status: %d", resp.StatusCode),
		}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// For now, just validate connectivity
	// In a real implementation, you would parse the response for version info
	return &PlatformValidationResult{
		Valid:      true,
		APIVersion: "unknown", // Would be parsed from response
	}, nil
}
