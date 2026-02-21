package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/openshift-online/regional-cli/internal/aws"
	"github.com/openshift-online/regional-cli/pkg/lambda/deployer"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/spf13/cobra"
)

const (
	defaultFunctionName      = "rosa-oidc-provisioner"
	defaultExecutionRoleName = "rosa-oidc-provisioner-execution"
	defaultMemorySize        = 128
	defaultTimeout           = 60
)

var (
	functionName      string
	executionRoleName string
	clmServiceRoleARN string
	sourceAccountID   string
)

// NewSetupAccountCommand creates the setup-account command
func NewSetupAccountCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup-account",
		Short: "Deploy OIDC provisioner Lambda to customer AWS account",
		Long: `Deploys the OIDC provisioner Lambda function that creates OIDC providers
for cluster authentication. This command:
  - Creates Lambda execution IAM role with minimal permissions
  - Builds and deploys the OIDC provisioner Lambda function
  - Configures CloudWatch Logs with 90-day retention
  - Optionally adds resource policy for CLM invocation`,
		RunE: runSetupAccount,
	}

	// Command-specific flags
	cmd.Flags().StringVar(&functionName, "function-name", defaultFunctionName, "Lambda function name")
	cmd.Flags().StringVar(&executionRoleName, "execution-role-name", defaultExecutionRoleName, "Lambda execution role name")
	cmd.Flags().StringVar(&clmServiceRoleARN, "clm-service-role-arn", "", "CLM service role ARN for resource policy")
	cmd.Flags().StringVar(&sourceAccountID, "source-account-id", "", "Source account ID for resource policy")

	return cmd
}

func runSetupAccount(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	profile, region, verbose, _ := getGlobalFlags()

	if verbose {
		fmt.Println("Setting up customer AWS account for ROSA...")
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

	// Create AWS service clients
	lambdaClient := aws.NewLambdaClient(awsConfig)
	iamClient := aws.NewIAMClient(awsConfig)
	cwLogsClient := aws.NewCloudWatchLogsClient(awsConfig)

	// Determine source directory for Lambda function
	// In production, this would be embedded or downloaded
	// For now, use relative path
	sourceDir := filepath.Join("pkg", "lambda", "functions", "oidc-provisioner")

	// Create deployment config
	deployConfig := deployer.DeploymentConfig{
		FunctionName:      functionName,
		ExecutionRoleName: executionRoleName,
		SourceDir:         sourceDir,
		CLMServiceRoleARN: clmServiceRoleARN,
		SourceAccountID:   sourceAccountID,
		Runtime:           lambdaTypes.RuntimeProvidedal2023,
		MemorySize:        defaultMemorySize,
		Timeout:           defaultTimeout,
		Architecture:      lambdaTypes.ArchitectureX8664,
		Tags: map[string]string{
			"rosa:component": "oidc-provisioner",
			"rosa:managed":   "true",
		},
	}

	// Create deployer
	lambdaDeployer := deployer.NewDeployer(lambdaClient, iamClient, cwLogsClient, deployConfig)

	// Deploy Lambda function
	fmt.Println("Deploying OIDC provisioner Lambda function...")

	result, err := lambdaDeployer.Deploy(ctx)
	if err != nil {
		fmt.Printf("✗ Deployment failed\n")
		return err
	}

	// Display results
	fmt.Printf("✓ Lambda function %s: %s\n", result.Status, result.FunctionName)
	if verbose {
		fmt.Printf("  Function ARN: %s\n", result.FunctionARN)
		fmt.Printf("  Execution Role: %s\n", result.ExecutionRole)
		fmt.Printf("  Log Group: %s\n", result.LogGroupName)
		fmt.Printf("  Package Size: %d bytes\n", result.PackageSize)
		fmt.Printf("  Package Checksum: %s\n", result.PackageChecksum)
	}

	if result.Status == "created" {
		fmt.Println("✓ IAM execution role created")
		fmt.Println("✓ CloudWatch Log Group created")
	} else {
		fmt.Println("✓ Lambda function updated")
	}

	if clmServiceRoleARN != "" && sourceAccountID != "" {
		fmt.Println("✓ Resource policy configured for CLM invocation")
	}

	fmt.Printf("\nSetup complete. Lambda function deployed: %s\n", result.FunctionARN)
	fmt.Println("Your AWS account is now configured for ROSA cluster provisioning.")

	return nil
}
