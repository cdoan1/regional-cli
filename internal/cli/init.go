package cli

import (
	"context"
	"fmt"

	"github.com/openshift-online/regional-cli/internal/aws"
	"github.com/openshift-online/regional-cli/internal/validator"
	"github.com/spf13/cobra"
)

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Validate AWS credentials and Platform API connectivity",
		Long: `Validates that:
  - AWS credentials are configured and valid
  - AWS region is set and supported
  - Platform API is reachable (if URL is provided)`,
		RunE: runInit,
	}

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	profile, region, verbose, platformAPIURL := getGlobalFlags()

	if verbose {
		fmt.Println("Validating AWS credentials and configuration...")
	}

	// Create AWS config
	awsConfig, err := aws.NewConfig(ctx, aws.ClientConfig{
		Profile: profile,
		Region:  region,
	})
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// If region not specified via flag, get it from config
	if region == "" {
		region = awsConfig.Region
	}

	// Validate AWS credentials
	stsClient := aws.NewSTSClient(awsConfig)
	awsValidator := validator.NewAWSValidator(stsClient, region)

	awsResult, err := awsValidator.Validate(ctx)
	if err != nil {
		fmt.Printf("✗ AWS credentials validation failed\n")
		return err
	}

	if !awsResult.Valid {
		fmt.Printf("✗ AWS validation failed: %s\n", awsResult.ErrorMessage)
		return fmt.Errorf("AWS validation failed")
	}

	fmt.Printf("✓ AWS credentials valid\n")
	if verbose {
		fmt.Printf("  Account ID: %s\n", awsResult.AccountID)
		fmt.Printf("  User ARN: %s\n", awsResult.UserARN)
		fmt.Printf("  Region: %s\n", awsResult.Region)
	}

	// Validate Platform API connectivity (if URL provided)
	if platformAPIURL != "" {
		if verbose {
			fmt.Printf("Validating Platform API connectivity to %s...\n", platformAPIURL)
		}

		platformValidator := validator.NewPlatformValidator(platformAPIURL, awsConfig)
		platformResult, err := platformValidator.Validate(ctx)

		if err != nil {
			fmt.Printf("✗ Platform API validation failed\n")
			fmt.Printf("  Error: %s\n", platformResult.ErrorMessage)
			return err
		}

		if !platformResult.Valid {
			fmt.Printf("✗ Platform API validation failed: %s\n", platformResult.ErrorMessage)
			return fmt.Errorf("Platform API validation failed")
		}

		fmt.Printf("✓ Platform API reachable\n")
		if verbose {
			fmt.Printf("  Base URL: %s\n", platformAPIURL)
			fmt.Printf("  Live endpoint: %s/prod/v0/live\n", platformAPIURL)
			fmt.Printf("  Response: %s\n", platformResult.APIVersion)
		}
	} else {
		if verbose {
			fmt.Println("Skipping Platform API validation (no URL provided)")
		}
	}

	fmt.Println("\nValidation complete. Your environment is configured correctly.")
	return nil
}
