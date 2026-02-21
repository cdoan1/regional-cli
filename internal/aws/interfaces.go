package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// LambdaAPI defines testable Lambda operations
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
	Invoke(ctx context.Context, params *lambda.InvokeInput,
		optFns ...func(*lambda.Options)) (*lambda.InvokeOutput, error)
	TagResource(ctx context.Context, params *lambda.TagResourceInput,
		optFns ...func(*lambda.Options)) (*lambda.TagResourceOutput, error)
}

// IAMAPI defines testable IAM operations
type IAMAPI interface {
	CreateRole(ctx context.Context, params *iam.CreateRoleInput,
		optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error)
	GetRole(ctx context.Context, params *iam.GetRoleInput,
		optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error)
	PutRolePolicy(ctx context.Context, params *iam.PutRolePolicyInput,
		optFns ...func(*iam.Options)) (*iam.PutRolePolicyOutput, error)
	CreateOpenIDConnectProvider(ctx context.Context, params *iam.CreateOpenIDConnectProviderInput,
		optFns ...func(*iam.Options)) (*iam.CreateOpenIDConnectProviderOutput, error)
	GetOpenIDConnectProvider(ctx context.Context, params *iam.GetOpenIDConnectProviderInput,
		optFns ...func(*iam.Options)) (*iam.GetOpenIDConnectProviderOutput, error)
	TagOpenIDConnectProvider(ctx context.Context, params *iam.TagOpenIDConnectProviderInput,
		optFns ...func(*iam.Options)) (*iam.TagOpenIDConnectProviderOutput, error)
	ListOpenIDConnectProviders(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput,
		optFns ...func(*iam.Options)) (*iam.ListOpenIDConnectProvidersOutput, error)
}

// STSAPI defines testable STS operations
type STSAPI interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput,
		optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

// CloudWatchLogsAPI defines testable CloudWatch Logs operations
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
