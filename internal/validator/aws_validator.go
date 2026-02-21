package validator

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// STSAPI defines the STS operations needed for validation
type STSAPI interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput,
		optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

// AWSValidator validates AWS credentials and configuration
type AWSValidator struct {
	stsClient STSAPI
	region    string
}

// NewAWSValidator creates a new AWS validator
func NewAWSValidator(stsClient STSAPI, region string) *AWSValidator {
	return &AWSValidator{
		stsClient: stsClient,
		region:    region,
	}
}

// ValidationResult holds the result of AWS validation
type ValidationResult struct {
	Valid         bool
	AccountID     string
	UserARN       string
	Region        string
	ErrorMessage  string
}

// Validate validates AWS credentials and returns account information
func (v *AWSValidator) Validate(ctx context.Context) (*ValidationResult, error) {
	// Validate credentials by calling GetCallerIdentity
	output, err := v.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return &ValidationResult{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("Failed to validate AWS credentials: %v", err),
		}, err
	}

	// Validate region
	if v.region == "" {
		return &ValidationResult{
			Valid:        false,
			ErrorMessage: "AWS region is not configured",
		}, fmt.Errorf("region not configured")
	}

	// Check if region is in supported list
	if !isSupportedRegion(v.region) {
		return &ValidationResult{
			Valid:        false,
			Region:       v.region,
			ErrorMessage: fmt.Sprintf("AWS region '%s' is not supported", v.region),
		}, fmt.Errorf("unsupported region: %s", v.region)
	}

	return &ValidationResult{
		Valid:     true,
		AccountID: aws.ToString(output.Account),
		UserARN:   aws.ToString(output.Arn),
		Region:    v.region,
	}, nil
}

// isSupportedRegion checks if the region is in the supported list
func isSupportedRegion(region string) bool {
	supportedRegions := []string{
		"us-east-1",
		"us-east-2",
		"us-west-1",
		"us-west-2",
		"eu-west-1",
		"eu-west-2",
		"eu-west-3",
		"eu-central-1",
		"eu-north-1",
		"ap-southeast-1",
		"ap-southeast-2",
		"ap-northeast-1",
		"ap-northeast-2",
		"ap-south-1",
		"sa-east-1",
		"ca-central-1",
	}

	for _, supported := range supportedRegions {
		if region == supported {
			return true
		}
	}
	return false
}
