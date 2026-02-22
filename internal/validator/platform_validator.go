package validator

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

// PlatformValidator validates Platform API connectivity
type PlatformValidator struct {
	apiURL     string
	awsConfig  aws.Config
	httpClient *http.Client
}

// NewPlatformValidator creates a new Platform API validator
func NewPlatformValidator(apiURL string, awsConfig aws.Config) *PlatformValidator {
	return &PlatformValidator{
		apiURL:    apiURL,
		awsConfig: awsConfig,
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

// extractRegionFromURL extracts the AWS region from an API Gateway URL
func extractRegionFromURL(url string) string {
	// Match pattern like: https://xxx.execute-api.REGION.amazonaws.com
	re := regexp.MustCompile(`execute-api\.([a-z0-9-]+)\.amazonaws\.com`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// Validate validates Platform API connectivity
func (v *PlatformValidator) Validate(ctx context.Context) (*PlatformValidationResult, error) {
	if v.apiURL == "" {
		return &PlatformValidationResult{
			Valid:        false,
			ErrorMessage: "Platform API URL is not configured",
		}, fmt.Errorf("API URL not configured")
	}

	// Extract region from API URL for SigV4 signing
	apiRegion := extractRegionFromURL(v.apiURL)
	if apiRegion == "" {
		// Fall back to config region if we can't extract it
		apiRegion = v.awsConfig.Region
	}

	// Use the correct live endpoint
	liveURL := v.apiURL + "/prod/v0/live"

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", liveURL, nil)
	if err != nil {
		return &PlatformValidationResult{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("Failed to create request to %s: %v", liveURL, err),
		}, err
	}

	// Sign request with AWS SigV4 using the API's region
	credentials, err := v.awsConfig.Credentials.Retrieve(ctx)
	if err != nil {
		return &PlatformValidationResult{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("Failed to retrieve AWS credentials for signing: %v", err),
		}, err
	}

	// Calculate payload hash for empty body (GET request)
	payloadHash := fmt.Sprintf("%x", sha256.Sum256([]byte{}))

	signer := v4.NewSigner()
	err = signer.SignHTTP(ctx, credentials, req, payloadHash, "execute-api", apiRegion, time.Now())
	if err != nil {
		return &PlatformValidationResult{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("Failed to sign request: %v", err),
		}, err
	}

	// Execute request
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return &PlatformValidationResult{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("Failed to connect to %s: %v", liveURL, err),
		}, err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		// Read response body for more details
		body, _ := io.ReadAll(resp.Body)
		return &PlatformValidationResult{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("GET %s returned status: %d, body: %s", liveURL, resp.StatusCode, string(body)),
		}, fmt.Errorf("GET %s returned status code: %d", liveURL, resp.StatusCode)
	}

	// Read and parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &PlatformValidationResult{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("Failed to read response: %v", err),
		}, err
	}

	// For now, just validate we got a response
	// In a real implementation, you would parse JSON for version info
	return &PlatformValidationResult{
		Valid:      true,
		APIVersion: string(body), // Contains {"status":"ok"}
	}, nil
}
