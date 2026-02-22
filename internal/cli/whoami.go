package cli

import (
	"context"
	"fmt"

	"github.com/openshift-online/regional-cli/internal/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
)

// NewWhoamiCommand creates the whoami command
func NewWhoamiCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Display current AWS identity information",
		Long:  `Display the current AWS STS caller identity including UserId and Account.`,
		RunE:  runWhoami,
	}

	return cmd
}

func runWhoami(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	profile, region, _, _ := getGlobalFlags()

	// Create AWS config
	awsConfig, err := aws.NewConfig(ctx, aws.ClientConfig{
		Profile: profile,
		Region:  region,
	})
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Get caller identity
	stsClient := aws.NewSTSClient(awsConfig)
	output, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to get caller identity: %w", err)
	}

	// Display identity information
	fmt.Printf("UserId:  %s\n", awssdk.ToString(output.UserId))
	fmt.Printf("Account: %s\n", awssdk.ToString(output.Account))
	fmt.Printf("Arn:     %s\n", awssdk.ToString(output.Arn))

	return nil
}
