package deployer

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations
type mockLambdaClient struct {
	createFunctionFunc        func(ctx context.Context, params *lambda.CreateFunctionInput, optFns ...func(*lambda.Options)) (*lambda.CreateFunctionOutput, error)
	updateFunctionCodeFunc    func(ctx context.Context, params *lambda.UpdateFunctionCodeInput, optFns ...func(*lambda.Options)) (*lambda.UpdateFunctionCodeOutput, error)
	updateFunctionConfigFunc  func(ctx context.Context, params *lambda.UpdateFunctionConfigurationInput, optFns ...func(*lambda.Options)) (*lambda.UpdateFunctionConfigurationOutput, error)
	getFunctionFunc           func(ctx context.Context, params *lambda.GetFunctionInput, optFns ...func(*lambda.Options)) (*lambda.GetFunctionOutput, error)
	addPermissionFunc         func(ctx context.Context, params *lambda.AddPermissionInput, optFns ...func(*lambda.Options)) (*lambda.AddPermissionOutput, error)
	tagResourceFunc           func(ctx context.Context, params *lambda.TagResourceInput, optFns ...func(*lambda.Options)) (*lambda.TagResourceOutput, error)
}

func (m *mockLambdaClient) CreateFunction(ctx context.Context, params *lambda.CreateFunctionInput, optFns ...func(*lambda.Options)) (*lambda.CreateFunctionOutput, error) {
	if m.createFunctionFunc != nil {
		return m.createFunctionFunc(ctx, params, optFns...)
	}
	return &lambda.CreateFunctionOutput{}, nil
}

func (m *mockLambdaClient) UpdateFunctionCode(ctx context.Context, params *lambda.UpdateFunctionCodeInput, optFns ...func(*lambda.Options)) (*lambda.UpdateFunctionCodeOutput, error) {
	if m.updateFunctionCodeFunc != nil {
		return m.updateFunctionCodeFunc(ctx, params, optFns...)
	}
	return &lambda.UpdateFunctionCodeOutput{}, nil
}

func (m *mockLambdaClient) UpdateFunctionConfiguration(ctx context.Context, params *lambda.UpdateFunctionConfigurationInput, optFns ...func(*lambda.Options)) (*lambda.UpdateFunctionConfigurationOutput, error) {
	if m.updateFunctionConfigFunc != nil {
		return m.updateFunctionConfigFunc(ctx, params, optFns...)
	}
	return &lambda.UpdateFunctionConfigurationOutput{}, nil
}

func (m *mockLambdaClient) GetFunction(ctx context.Context, params *lambda.GetFunctionInput, optFns ...func(*lambda.Options)) (*lambda.GetFunctionOutput, error) {
	if m.getFunctionFunc != nil {
		return m.getFunctionFunc(ctx, params, optFns...)
	}
	return &lambda.GetFunctionOutput{}, nil
}

func (m *mockLambdaClient) AddPermission(ctx context.Context, params *lambda.AddPermissionInput, optFns ...func(*lambda.Options)) (*lambda.AddPermissionOutput, error) {
	if m.addPermissionFunc != nil {
		return m.addPermissionFunc(ctx, params, optFns...)
	}
	return &lambda.AddPermissionOutput{}, nil
}

func (m *mockLambdaClient) TagResource(ctx context.Context, params *lambda.TagResourceInput, optFns ...func(*lambda.Options)) (*lambda.TagResourceOutput, error) {
	if m.tagResourceFunc != nil {
		return m.tagResourceFunc(ctx, params, optFns...)
	}
	return &lambda.TagResourceOutput{}, nil
}

type mockIAMClient struct {
	createRoleFunc    func(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error)
	getRoleFunc       func(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error)
	putRolePolicyFunc func(ctx context.Context, params *iam.PutRolePolicyInput, optFns ...func(*iam.Options)) (*iam.PutRolePolicyOutput, error)
}

func (m *mockIAMClient) CreateRole(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error) {
	if m.createRoleFunc != nil {
		return m.createRoleFunc(ctx, params, optFns...)
	}
	return &iam.CreateRoleOutput{}, nil
}

func (m *mockIAMClient) GetRole(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
	if m.getRoleFunc != nil {
		return m.getRoleFunc(ctx, params, optFns...)
	}
	return &iam.GetRoleOutput{}, nil
}

func (m *mockIAMClient) PutRolePolicy(ctx context.Context, params *iam.PutRolePolicyInput, optFns ...func(*iam.Options)) (*iam.PutRolePolicyOutput, error) {
	if m.putRolePolicyFunc != nil {
		return m.putRolePolicyFunc(ctx, params, optFns...)
	}
	return &iam.PutRolePolicyOutput{}, nil
}

type mockCloudWatchLogsClient struct {
	createLogGroupFunc      func(ctx context.Context, params *cloudwatchlogs.CreateLogGroupInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.CreateLogGroupOutput, error)
	describeLogGroupsFunc   func(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error)
	putRetentionPolicyFunc  func(ctx context.Context, params *cloudwatchlogs.PutRetentionPolicyInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutRetentionPolicyOutput, error)
	tagLogGroupFunc         func(ctx context.Context, params *cloudwatchlogs.TagLogGroupInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.TagLogGroupOutput, error)
}

func (m *mockCloudWatchLogsClient) CreateLogGroup(ctx context.Context, params *cloudwatchlogs.CreateLogGroupInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.CreateLogGroupOutput, error) {
	if m.createLogGroupFunc != nil {
		return m.createLogGroupFunc(ctx, params, optFns...)
	}
	return &cloudwatchlogs.CreateLogGroupOutput{}, nil
}

func (m *mockCloudWatchLogsClient) DescribeLogGroups(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	if m.describeLogGroupsFunc != nil {
		return m.describeLogGroupsFunc(ctx, params, optFns...)
	}
	return &cloudwatchlogs.DescribeLogGroupsOutput{}, nil
}

func (m *mockCloudWatchLogsClient) PutRetentionPolicy(ctx context.Context, params *cloudwatchlogs.PutRetentionPolicyInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutRetentionPolicyOutput, error) {
	if m.putRetentionPolicyFunc != nil {
		return m.putRetentionPolicyFunc(ctx, params, optFns...)
	}
	return &cloudwatchlogs.PutRetentionPolicyOutput{}, nil
}

func (m *mockCloudWatchLogsClient) TagLogGroup(ctx context.Context, params *cloudwatchlogs.TagLogGroupInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.TagLogGroupOutput, error) {
	if m.tagLogGroupFunc != nil {
		return m.tagLogGroupFunc(ctx, params, optFns...)
	}
	return &cloudwatchlogs.TagLogGroupOutput{}, nil
}

func TestDeploy_CreateNewFunction(t *testing.T) {
	ctx := context.Background()
	roleARN := "arn:aws:iam::123456789012:role/test-role"
	functionARN := "arn:aws:lambda:us-east-1:123456789012:function:test-function"

	mockLambda := &mockLambdaClient{
		getFunctionFunc: func(ctx context.Context, params *lambda.GetFunctionInput, optFns ...func(*lambda.Options)) (*lambda.GetFunctionOutput, error) {
			// Function doesn't exist
			return nil, &lambdaTypes.ResourceNotFoundException{}
		},
		createFunctionFunc: func(ctx context.Context, params *lambda.CreateFunctionInput, optFns ...func(*lambda.Options)) (*lambda.CreateFunctionOutput, error) {
			assert.Equal(t, "test-function", *params.FunctionName)
			assert.Equal(t, roleARN, *params.Role)
			assert.NotEmpty(t, params.Code.ZipFile)
			return &lambda.CreateFunctionOutput{
				FunctionArn: aws.String(functionARN),
			}, nil
		},
		tagResourceFunc: func(ctx context.Context, params *lambda.TagResourceInput, optFns ...func(*lambda.Options)) (*lambda.TagResourceOutput, error) {
			return &lambda.TagResourceOutput{}, nil
		},
	}

	mockIAM := &mockIAMClient{
		getRoleFunc: func(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
			return &iam.GetRoleOutput{
				Role: &iamTypes.Role{
					Arn: aws.String(roleARN),
				},
			}, nil
		},
	}

	mockCWLogs := &mockCloudWatchLogsClient{
		describeLogGroupsFunc: func(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
			return &cloudwatchlogs.DescribeLogGroupsOutput{
				LogGroups: []cwTypes.LogGroup{},
			}, nil
		},
		createLogGroupFunc: func(ctx context.Context, params *cloudwatchlogs.CreateLogGroupInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.CreateLogGroupOutput, error) {
			return &cloudwatchlogs.CreateLogGroupOutput{}, nil
		},
		putRetentionPolicyFunc: func(ctx context.Context, params *cloudwatchlogs.PutRetentionPolicyInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutRetentionPolicyOutput, error) {
			assert.Equal(t, int32(90), *params.RetentionInDays)
			return &cloudwatchlogs.PutRetentionPolicyOutput{}, nil
		},
	}

	config := DeploymentConfig{
		FunctionName:      "test-function",
		ExecutionRoleName: "test-role",
		SourceDir:         "../functions/oidc-provisioner",
		Runtime:           lambdaTypes.RuntimeProvidedal2023,
		MemorySize:        128,
		Timeout:           60,
		Architecture:      lambdaTypes.ArchitectureX8664,
		Tags: map[string]string{
			"Environment": "test",
		},
	}

	deployer := NewDeployer(mockLambda, mockIAM, mockCWLogs, config)
	result, err := deployer.Deploy(ctx)

	require.NoError(t, err)
	assert.Equal(t, functionARN, result.FunctionARN)
	assert.Equal(t, "test-function", result.FunctionName)
	assert.Equal(t, "created", result.Status)
	assert.Greater(t, result.PackageSize, 0)
	assert.NotEmpty(t, result.PackageChecksum)
}

func TestDeploy_UpdateExistingFunction(t *testing.T) {
	ctx := context.Background()
	roleARN := "arn:aws:iam::123456789012:role/test-role"
	functionARN := "arn:aws:lambda:us-east-1:123456789012:function:test-function"

	mockLambda := &mockLambdaClient{
		getFunctionFunc: func(ctx context.Context, params *lambda.GetFunctionInput, optFns ...func(*lambda.Options)) (*lambda.GetFunctionOutput, error) {
			// Function exists
			return &lambda.GetFunctionOutput{
				Configuration: &lambdaTypes.FunctionConfiguration{
					FunctionArn: aws.String(functionARN),
				},
			}, nil
		},
		updateFunctionCodeFunc: func(ctx context.Context, params *lambda.UpdateFunctionCodeInput, optFns ...func(*lambda.Options)) (*lambda.UpdateFunctionCodeOutput, error) {
			assert.NotEmpty(t, params.ZipFile)
			return &lambda.UpdateFunctionCodeOutput{}, nil
		},
		updateFunctionConfigFunc: func(ctx context.Context, params *lambda.UpdateFunctionConfigurationInput, optFns ...func(*lambda.Options)) (*lambda.UpdateFunctionConfigurationOutput, error) {
			assert.Equal(t, "test-function", *params.FunctionName)
			return &lambda.UpdateFunctionConfigurationOutput{}, nil
		},
		tagResourceFunc: func(ctx context.Context, params *lambda.TagResourceInput, optFns ...func(*lambda.Options)) (*lambda.TagResourceOutput, error) {
			return &lambda.TagResourceOutput{}, nil
		},
	}

	mockIAM := &mockIAMClient{
		getRoleFunc: func(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
			return &iam.GetRoleOutput{
				Role: &iamTypes.Role{
					Arn: aws.String(roleARN),
				},
			}, nil
		},
	}

	mockCWLogs := &mockCloudWatchLogsClient{
		describeLogGroupsFunc: func(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
			return &cloudwatchlogs.DescribeLogGroupsOutput{
				LogGroups: []cwTypes.LogGroup{
					{LogGroupName: aws.String("/aws/lambda/test-function")},
				},
			}, nil
		},
	}

	config := DeploymentConfig{
		FunctionName:      "test-function",
		ExecutionRoleName: "test-role",
		SourceDir:         "../functions/oidc-provisioner",
		Runtime:           lambdaTypes.RuntimeProvidedal2023,
		MemorySize:        128,
		Timeout:           60,
		Architecture:      lambdaTypes.ArchitectureX8664,
		Tags: map[string]string{
			"Environment": "test",
		},
	}

	deployer := NewDeployer(mockLambda, mockIAM, mockCWLogs, config)
	result, err := deployer.Deploy(ctx)

	require.NoError(t, err)
	assert.Equal(t, functionARN, result.FunctionARN)
	assert.Equal(t, "updated", result.Status)
}

func TestEnsureExecutionRole_CreateNewRole(t *testing.T) {
	ctx := context.Background()
	roleName := "test-role"
	roleARN := "arn:aws:iam::123456789012:role/test-role"

	mockIAM := &mockIAMClient{
		getRoleFunc: func(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
			// Role doesn't exist
			return nil, &iamTypes.NoSuchEntityException{}
		},
		createRoleFunc: func(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error) {
			assert.Equal(t, roleName, *params.RoleName)
			assert.NotEmpty(t, *params.AssumeRolePolicyDocument)
			return &iam.CreateRoleOutput{
				Role: &iamTypes.Role{
					Arn: aws.String(roleARN),
				},
			}, nil
		},
		putRolePolicyFunc: func(ctx context.Context, params *iam.PutRolePolicyInput, optFns ...func(*iam.Options)) (*iam.PutRolePolicyOutput, error) {
			assert.Equal(t, roleName, *params.RoleName)
			assert.NotEmpty(t, *params.PolicyDocument)
			return &iam.PutRolePolicyOutput{}, nil
		},
	}

	config := DeploymentConfig{
		ExecutionRoleName: roleName,
	}

	deployer := NewDeployer(nil, mockIAM, nil, config)
	arn, err := deployer.ensureExecutionRole(ctx)

	require.NoError(t, err)
	assert.Equal(t, roleARN, arn)
}

func TestEnsureExecutionRole_UseExistingRole(t *testing.T) {
	ctx := context.Background()
	roleName := "test-role"
	roleARN := "arn:aws:iam::123456789012:role/test-role"

	mockIAM := &mockIAMClient{
		getRoleFunc: func(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
			return &iam.GetRoleOutput{
				Role: &iamTypes.Role{
					Arn: aws.String(roleARN),
				},
			}, nil
		},
	}

	config := DeploymentConfig{
		ExecutionRoleName: roleName,
	}

	deployer := NewDeployer(nil, mockIAM, nil, config)
	arn, err := deployer.ensureExecutionRole(ctx)

	require.NoError(t, err)
	assert.Equal(t, roleARN, arn)
}

func TestEnsureExecutionRole_Error(t *testing.T) {
	ctx := context.Background()

	mockIAM := &mockIAMClient{
		getRoleFunc: func(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
			return nil, errors.New("unexpected error")
		},
	}

	config := DeploymentConfig{
		ExecutionRoleName: "test-role",
	}

	deployer := NewDeployer(nil, mockIAM, nil, config)
	_, err := deployer.ensureExecutionRole(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check if role exists")
}

func TestEnsureLogGroup(t *testing.T) {
	ctx := context.Background()
	logGroupName := "/aws/lambda/test-function"

	mockCWLogs := &mockCloudWatchLogsClient{
		describeLogGroupsFunc: func(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
			return &cloudwatchlogs.DescribeLogGroupsOutput{
				LogGroups: []cwTypes.LogGroup{},
			}, nil
		},
		createLogGroupFunc: func(ctx context.Context, params *cloudwatchlogs.CreateLogGroupInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.CreateLogGroupOutput, error) {
			assert.Equal(t, logGroupName, *params.LogGroupName)
			return &cloudwatchlogs.CreateLogGroupOutput{}, nil
		},
		putRetentionPolicyFunc: func(ctx context.Context, params *cloudwatchlogs.PutRetentionPolicyInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutRetentionPolicyOutput, error) {
			assert.Equal(t, logGroupName, *params.LogGroupName)
			assert.Equal(t, int32(90), *params.RetentionInDays)
			return &cloudwatchlogs.PutRetentionPolicyOutput{}, nil
		},
	}

	config := DeploymentConfig{}
	deployer := NewDeployer(nil, nil, mockCWLogs, config)

	err := deployer.ensureLogGroup(ctx, logGroupName)
	assert.NoError(t, err)
}

func TestAddResourcePolicy(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                 string
		clmRoleARN          string
		sourceAccountID     string
		addPermissionError  error
		expectError         bool
	}{
		{
			name:            "successful permission addition",
			clmRoleARN:      "arn:aws:iam::123456789012:role/clm-role",
			sourceAccountID: "123456789012",
			expectError:     false,
		},
		{
			name:               "permission already exists",
			clmRoleARN:         "arn:aws:iam::123456789012:role/clm-role",
			sourceAccountID:    "123456789012",
			addPermissionError: &lambdaTypes.ResourceConflictException{},
			expectError:        false, // Should not error on conflict
		},
		{
			name:               "permission error",
			clmRoleARN:         "arn:aws:iam::123456789012:role/clm-role",
			sourceAccountID:    "123456789012",
			addPermissionError: errors.New("unexpected error"),
			expectError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLambda := &mockLambdaClient{
				addPermissionFunc: func(ctx context.Context, params *lambda.AddPermissionInput, optFns ...func(*lambda.Options)) (*lambda.AddPermissionOutput, error) {
					assert.Equal(t, "test-function", *params.FunctionName)
					assert.Equal(t, "AllowCLMInvoke", *params.StatementId)
					if tt.addPermissionError != nil {
						return nil, tt.addPermissionError
					}
					return &lambda.AddPermissionOutput{}, nil
				},
			}

			config := DeploymentConfig{
				FunctionName:      "test-function",
				CLMServiceRoleARN: tt.clmRoleARN,
				SourceAccountID:   tt.sourceAccountID,
			}

			deployer := NewDeployer(mockLambda, nil, nil, config)
			err := deployer.addResourcePolicy(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckFunctionExists(t *testing.T) {
	ctx := context.Background()
	functionARN := "arn:aws:lambda:us-east-1:123456789012:function:test-function"

	t.Run("function exists", func(t *testing.T) {
		mockLambda := &mockLambdaClient{
			getFunctionFunc: func(ctx context.Context, params *lambda.GetFunctionInput, optFns ...func(*lambda.Options)) (*lambda.GetFunctionOutput, error) {
				return &lambda.GetFunctionOutput{
					Configuration: &lambdaTypes.FunctionConfiguration{
						FunctionArn: aws.String(functionARN),
					},
				}, nil
			},
		}

		config := DeploymentConfig{FunctionName: "test-function"}
		deployer := NewDeployer(mockLambda, nil, nil, config)

		exists, output, err := deployer.checkFunctionExists(ctx)
		require.NoError(t, err)
		assert.True(t, exists)
		assert.NotNil(t, output)
	})

	t.Run("function does not exist", func(t *testing.T) {
		mockLambda := &mockLambdaClient{
			getFunctionFunc: func(ctx context.Context, params *lambda.GetFunctionInput, optFns ...func(*lambda.Options)) (*lambda.GetFunctionOutput, error) {
				return nil, &lambdaTypes.ResourceNotFoundException{}
			},
		}

		config := DeploymentConfig{FunctionName: "test-function"}
		deployer := NewDeployer(mockLambda, nil, nil, config)

		exists, output, err := deployer.checkFunctionExists(ctx)
		require.NoError(t, err)
		assert.False(t, exists)
		assert.Nil(t, output)
	})
}
