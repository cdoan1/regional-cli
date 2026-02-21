package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

func main() {
	// Initialize AWS SDK
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic("failed to load AWS config: " + err.Error())
	}

	// Create IAM client
	iamClient := iam.NewFromConfig(cfg)

	// Create handler
	handler := NewHandler(iamClient)

	// Start Lambda
	lambda.Start(handler.Handle)
}
