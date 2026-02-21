package deployer

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// AWS service interfaces (defined in internal/aws/interfaces.go, but redefined here for package independence)
type LambdaAPI interface {
	CreateFunction(ctx context.Context, params *lambda.CreateFunctionInput,
		optFns ...func(*lambda.Options)) (*lambda.CreateFunctionOutput, error)
	UpdateFunctionCode(ctx context.Context, params *lambda.UpdateFunctionCodeInput,
		optFns ...func(*lambda.Options)) (*lambda.UpdateFunctionCodeOutput, error)
	UpdateFunctionConfiguration(ctx context.Context, params *lambda.UpdateFunctionConfigurationInput,
		optFns ...func(*lambda.Options)) (*lambda.UpdateFunctionConfigurationOutput, error)
	GetFunction(ctx context.Context, params *lambda.GetFunctionInput,
		optFns ...func(*lambda.Options)) (*lambda.GetFunctionOutput, error)
	AddPermission(ctx context.Context, params *lambda.AddPermissionInput,
		optFns ...func(*lambda.Options)) (*lambda.AddPermissionOutput, error)
	TagResource(ctx context.Context, params *lambda.TagResourceInput,
		optFns ...func(*lambda.Options)) (*lambda.TagResourceOutput, error)
}

type IAMAPI interface {
	CreateRole(ctx context.Context, params *iam.CreateRoleInput,
		optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error)
	GetRole(ctx context.Context, params *iam.GetRoleInput,
		optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error)
	PutRolePolicy(ctx context.Context, params *iam.PutRolePolicyInput,
		optFns ...func(*iam.Options)) (*iam.PutRolePolicyOutput, error)
}

type CloudWatchLogsAPI interface {
	CreateLogGroup(ctx context.Context, params *cloudwatchlogs.CreateLogGroupInput,
		optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.CreateLogGroupOutput, error)
	DescribeLogGroups(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput,
		optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error)
	PutRetentionPolicy(ctx context.Context, params *cloudwatchlogs.PutRetentionPolicyInput,
		optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutRetentionPolicyOutput, error)
	TagLogGroup(ctx context.Context, params *cloudwatchlogs.TagLogGroupInput,
		optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.TagLogGroupOutput, error)
}

// DeploymentConfig holds configuration for Lambda deployment
type DeploymentConfig struct {
	FunctionName      string
	ExecutionRoleName string
	SourceDir         string
	CLMServiceRoleARN string // Optional: for resource-based policy
	SourceAccountID   string // Optional: for resource-based policy
	Runtime           lambdaTypes.Runtime
	MemorySize        int32
	Timeout           int32
	Architecture      lambdaTypes.Architecture
	Tags              map[string]string
}

// Deployer orchestrates Lambda deployment
type Deployer struct {
	lambdaClient LambdaAPI
	iamClient    IAMAPI
	cwLogsClient CloudWatchLogsAPI
	config       DeploymentConfig
}

// NewDeployer creates a new Lambda deployer
func NewDeployer(lambdaClient LambdaAPI, iamClient IAMAPI, cwLogsClient CloudWatchLogsAPI, config DeploymentConfig) *Deployer {
	return &Deployer{
		lambdaClient: lambdaClient,
		iamClient:    iamClient,
		cwLogsClient: cwLogsClient,
		config:       config,
	}
}

// DeploymentResult holds the result of a deployment
type DeploymentResult struct {
	FunctionARN     string
	FunctionName    string
	ExecutionRole   string
	LogGroupName    string
	Status          string // "created", "updated", "already_exists"
	PackageSize     int
	PackageChecksum string
}

// Deploy orchestrates the full Lambda deployment
func (d *Deployer) Deploy(ctx context.Context) (*DeploymentResult, error) {
	// Step 1: Ensure IAM execution role exists
	roleARN, err := d.ensureExecutionRole(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure execution role: %w", err)
	}

	// Step 2: Build Lambda package
	packageBuilder := NewPackageBuilder(d.config.SourceDir)
	zipData, checksum, err := packageBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build Lambda package: %w", err)
	}

	// Step 3: Check if Lambda function exists
	exists, existingFunc, err := d.checkFunctionExists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check if function exists: %w", err)
	}

	var functionARN string
	var status string

	if exists {
		// Update existing function
		functionARN = *existingFunc.Configuration.FunctionArn
		if err := d.updateFunction(ctx, zipData, roleARN); err != nil {
			return nil, fmt.Errorf("failed to update function: %w", err)
		}
		status = "updated"
	} else {
		// Create new function
		functionARN, err = d.createFunction(ctx, zipData, roleARN)
		if err != nil {
			return nil, fmt.Errorf("failed to create function: %w", err)
		}
		status = "created"
	}

	// Step 4: Add resource-based policy (if CLM service role ARN is provided)
	if d.config.CLMServiceRoleARN != "" && d.config.SourceAccountID != "" {
		if err := d.addResourcePolicy(ctx); err != nil {
			// Don't fail deployment if policy already exists
			fmt.Printf("Warning: failed to add resource policy: %v\n", err)
		}
	}

	// Step 5: Ensure CloudWatch Log Group exists
	logGroupName := fmt.Sprintf("/aws/lambda/%s", d.config.FunctionName)
	if err := d.ensureLogGroup(ctx, logGroupName); err != nil {
		// Don't fail deployment if log group creation fails
		fmt.Printf("Warning: failed to ensure log group: %v\n", err)
	}

	// Step 6: Tag Lambda function
	if len(d.config.Tags) > 0 {
		if err := d.tagFunction(ctx, functionARN); err != nil {
			fmt.Printf("Warning: failed to tag function: %v\n", err)
		}
	}

	return &DeploymentResult{
		FunctionARN:     functionARN,
		FunctionName:    d.config.FunctionName,
		ExecutionRole:   roleARN,
		LogGroupName:    logGroupName,
		Status:          status,
		PackageSize:     len(zipData),
		PackageChecksum: checksum,
	}, nil
}

// ensureExecutionRole creates or gets the Lambda execution role
func (d *Deployer) ensureExecutionRole(ctx context.Context) (string, error) {
	// Try to get existing role
	getOutput, err := d.iamClient.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(d.config.ExecutionRoleName),
	})

	if err == nil {
		// Role exists
		return *getOutput.Role.Arn, nil
	}

	// Check if error is "not found"
	var notFoundErr *iamTypes.NoSuchEntityException
	if !errors.As(err, &notFoundErr) {
		return "", fmt.Errorf("failed to check if role exists: %w", err)
	}

	// Role doesn't exist, create it
	trustPolicy, err := GenerateLambdaExecutionRoleTrustPolicy()
	if err != nil {
		return "", fmt.Errorf("failed to generate trust policy: %w", err)
	}

	createOutput, err := d.iamClient.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(d.config.ExecutionRoleName),
		AssumeRolePolicyDocument: aws.String(trustPolicy),
		Description:              aws.String("Execution role for ROSA OIDC provisioner Lambda"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create role: %w", err)
	}

	roleARN := *createOutput.Role.Arn

	// Attach inline permissions policy
	permissionsPolicy, err := GenerateOIDCProvisionerPermissionsPolicy()
	if err != nil {
		return "", fmt.Errorf("failed to generate permissions policy: %w", err)
	}

	_, err = d.iamClient.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		RoleName:       aws.String(d.config.ExecutionRoleName),
		PolicyName:     aws.String("OIDCProvisionerPermissions"),
		PolicyDocument: aws.String(permissionsPolicy),
	})
	if err != nil {
		return "", fmt.Errorf("failed to attach permissions policy: %w", err)
	}

	return roleARN, nil
}

// checkFunctionExists checks if the Lambda function already exists
func (d *Deployer) checkFunctionExists(ctx context.Context) (bool, *lambda.GetFunctionOutput, error) {
	output, err := d.lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(d.config.FunctionName),
	})

	if err != nil {
		var notFoundErr *lambdaTypes.ResourceNotFoundException
		if errors.As(err, &notFoundErr) {
			return false, nil, nil
		}
		return false, nil, err
	}

	return true, output, nil
}

// createFunction creates a new Lambda function
func (d *Deployer) createFunction(ctx context.Context, zipData []byte, roleARN string) (string, error) {
	output, err := d.lambdaClient.CreateFunction(ctx, &lambda.CreateFunctionInput{
		FunctionName: aws.String(d.config.FunctionName),
		Runtime:      d.config.Runtime,
		Role:         aws.String(roleARN),
		Handler:      aws.String("bootstrap"), // Required for custom runtime
		Code: &lambdaTypes.FunctionCode{
			ZipFile: zipData,
		},
		MemorySize:   aws.Int32(d.config.MemorySize),
		Timeout:      aws.Int32(d.config.Timeout),
		Architectures: []lambdaTypes.Architecture{d.config.Architecture},
		Description:  aws.String("ROSA OIDC provider provisioner"),
	})

	if err != nil {
		return "", err
	}

	return *output.FunctionArn, nil
}

// updateFunction updates an existing Lambda function
func (d *Deployer) updateFunction(ctx context.Context, zipData []byte, roleARN string) error {
	// Update code
	_, err := d.lambdaClient.UpdateFunctionCode(ctx, &lambda.UpdateFunctionCodeInput{
		FunctionName: aws.String(d.config.FunctionName),
		ZipFile:      zipData,
	})
	if err != nil {
		return fmt.Errorf("failed to update function code: %w", err)
	}

	// Update configuration
	_, err = d.lambdaClient.UpdateFunctionConfiguration(ctx, &lambda.UpdateFunctionConfigurationInput{
		FunctionName: aws.String(d.config.FunctionName),
		Runtime:      d.config.Runtime,
		Role:         aws.String(roleARN),
		Handler:      aws.String("bootstrap"),
		MemorySize:   aws.Int32(d.config.MemorySize),
		Timeout:      aws.Int32(d.config.Timeout),
	})
	if err != nil {
		return fmt.Errorf("failed to update function configuration: %w", err)
	}

	return nil
}

// addResourcePolicy adds a resource-based policy to allow CLM to invoke the Lambda
func (d *Deployer) addResourcePolicy(ctx context.Context) error {
	policy, err := GenerateLambdaResourcePolicy(d.config.CLMServiceRoleARN, d.config.SourceAccountID)
	if err != nil {
		return err
	}

	// Add permission (idempotent - will return error if already exists, which we ignore)
	_, err = d.lambdaClient.AddPermission(ctx, &lambda.AddPermissionInput{
		FunctionName: aws.String(d.config.FunctionName),
		StatementId:  aws.String("AllowCLMInvoke"),
		Action:       aws.String("lambda:InvokeFunction"),
		Principal:    aws.String("arn:aws:iam::" + d.config.SourceAccountID + ":root"),
		SourceArn:    aws.String(d.config.CLMServiceRoleARN),
	})

	if err != nil {
		// Check if permission already exists
		var resourceConflictErr *lambdaTypes.ResourceConflictException
		if errors.As(err, &resourceConflictErr) {
			// Permission already exists, not an error
			return nil
		}
		return err
	}

	_ = policy // Policy string generated but not directly used (AddPermission handles it)
	return nil
}

// ensureLogGroup ensures the CloudWatch Log Group exists with retention
func (d *Deployer) ensureLogGroup(ctx context.Context, logGroupName string) error {
	// Check if log group exists
	describeOutput, err := d.cwLogsClient.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(logGroupName),
	})

	if err == nil && len(describeOutput.LogGroups) > 0 {
		// Log group already exists
		for _, lg := range describeOutput.LogGroups {
			if *lg.LogGroupName == logGroupName {
				return nil // Already exists
			}
		}
	}

	// Create log group
	_, err = d.cwLogsClient.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(logGroupName),
	})

	if err != nil {
		var alreadyExistsErr *types.ResourceAlreadyExistsException
		if !errors.As(err, &alreadyExistsErr) {
			return fmt.Errorf("failed to create log group: %w", err)
		}
	}

	// Set retention policy (90 days)
	_, err = d.cwLogsClient.PutRetentionPolicy(ctx, &cloudwatchlogs.PutRetentionPolicyInput{
		LogGroupName:    aws.String(logGroupName),
		RetentionInDays: aws.Int32(90),
	})

	if err != nil {
		return fmt.Errorf("failed to set retention policy: %w", err)
	}

	// Tag log group
	if len(d.config.Tags) > 0 {
		tags := make(map[string]string)
		for k, v := range d.config.Tags {
			tags[k] = v
		}

		_, err = d.cwLogsClient.TagLogGroup(ctx, &cloudwatchlogs.TagLogGroupInput{
			LogGroupName: aws.String(logGroupName),
			Tags:         tags,
		})

		if err != nil {
			// Don't fail if tagging fails
			fmt.Printf("Warning: failed to tag log group: %v\n", err)
		}
	}

	return nil
}

// tagFunction tags the Lambda function
func (d *Deployer) tagFunction(ctx context.Context, functionARN string) error {
	_, err := d.lambdaClient.TagResource(ctx, &lambda.TagResourceInput{
		Resource: aws.String(functionARN),
		Tags:     d.config.Tags,
	})
	return err
}

// EncodeBase64 encodes data to base64 (utility for testing)
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
