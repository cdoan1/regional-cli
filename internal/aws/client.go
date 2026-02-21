package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// ClientConfig holds AWS client configuration options
type ClientConfig struct {
	Profile string
	Region  string
}

// NewConfig creates an AWS SDK v2 config from the provided options
func NewConfig(ctx context.Context, cfg ClientConfig) (aws.Config, error) {
	var opts []func(*config.LoadOptions) error

	if cfg.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(cfg.Profile))
	}

	if cfg.Region != "" {
		opts = append(opts, config.WithRegion(cfg.Region))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return awsCfg, nil
}

// NewLambdaClient creates a new Lambda client
func NewLambdaClient(cfg aws.Config) LambdaAPI {
	return lambda.NewFromConfig(cfg)
}

// NewIAMClient creates a new IAM client
func NewIAMClient(cfg aws.Config) IAMAPI {
	return iam.NewFromConfig(cfg)
}

// NewSTSClient creates a new STS client
func NewSTSClient(cfg aws.Config) STSAPI {
	return sts.NewFromConfig(cfg)
}

// NewCloudWatchLogsClient creates a new CloudWatch Logs client
func NewCloudWatchLogsClient(cfg aws.Config) CloudWatchLogsAPI {
	return cloudwatchlogs.NewFromConfig(cfg)
}
