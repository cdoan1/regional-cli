package validator

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSTSClient struct {
	getCallerIdentityFunc func(ctx context.Context, params *sts.GetCallerIdentityInput,
		optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

func (m *mockSTSClient) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput,
	optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	if m.getCallerIdentityFunc != nil {
		return m.getCallerIdentityFunc(ctx, params, optFns...)
	}
	return &sts.GetCallerIdentityOutput{}, nil
}

func TestValidate_Success(t *testing.T) {
	ctx := context.Background()
	expectedAccountID := "123456789012"
	expectedUserARN := "arn:aws:iam::123456789012:user/test-user"

	mockSTS := &mockSTSClient{
		getCallerIdentityFunc: func(ctx context.Context, params *sts.GetCallerIdentityInput,
			optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
			return &sts.GetCallerIdentityOutput{
				Account: aws.String(expectedAccountID),
				Arn:     aws.String(expectedUserARN),
			}, nil
		},
	}

	validator := NewAWSValidator(mockSTS, "us-east-1")
	result, err := validator.Validate(ctx)

	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, expectedAccountID, result.AccountID)
	assert.Equal(t, expectedUserARN, result.UserARN)
	assert.Equal(t, "us-east-1", result.Region)
	assert.Empty(t, result.ErrorMessage)
}

func TestValidate_InvalidCredentials(t *testing.T) {
	ctx := context.Background()

	mockSTS := &mockSTSClient{
		getCallerIdentityFunc: func(ctx context.Context, params *sts.GetCallerIdentityInput,
			optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
			return nil, errors.New("invalid credentials")
		},
	}

	validator := NewAWSValidator(mockSTS, "us-east-1")
	result, err := validator.Validate(ctx)

	assert.Error(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessage, "Failed to validate AWS credentials")
}

func TestValidate_NoRegion(t *testing.T) {
	ctx := context.Background()

	mockSTS := &mockSTSClient{
		getCallerIdentityFunc: func(ctx context.Context, params *sts.GetCallerIdentityInput,
			optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
			return &sts.GetCallerIdentityOutput{
				Account: aws.String("123456789012"),
				Arn:     aws.String("arn:aws:iam::123456789012:user/test-user"),
			}, nil
		},
	}

	validator := NewAWSValidator(mockSTS, "")
	result, err := validator.Validate(ctx)

	assert.Error(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessage, "AWS region is not configured")
}

func TestValidate_UnsupportedRegion(t *testing.T) {
	ctx := context.Background()

	mockSTS := &mockSTSClient{
		getCallerIdentityFunc: func(ctx context.Context, params *sts.GetCallerIdentityInput,
			optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
			return &sts.GetCallerIdentityOutput{
				Account: aws.String("123456789012"),
				Arn:     aws.String("arn:aws:iam::123456789012:user/test-user"),
			}, nil
		},
	}

	validator := NewAWSValidator(mockSTS, "unsupported-region")
	result, err := validator.Validate(ctx)

	assert.Error(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.ErrorMessage, "not supported")
}

func TestIsSupportedRegion(t *testing.T) {
	tests := []struct {
		region   string
		expected bool
	}{
		{"us-east-1", true},
		{"us-west-2", true},
		{"eu-west-1", true},
		{"ap-southeast-1", true},
		{"unsupported-region", false},
		{"us-east-3", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			result := isSupportedRegion(tt.region)
			assert.Equal(t, tt.expected, result)
		})
	}
}
