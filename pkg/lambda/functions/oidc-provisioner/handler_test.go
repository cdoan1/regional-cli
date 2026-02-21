package main

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockIAMClient is a mock implementation of IAMAPI
type mockIAMClient struct {
	createOIDCProviderFunc func(ctx context.Context, params *iam.CreateOpenIDConnectProviderInput,
		optFns ...func(*iam.Options)) (*iam.CreateOpenIDConnectProviderOutput, error)
	getOIDCProviderFunc func(ctx context.Context, params *iam.GetOpenIDConnectProviderInput,
		optFns ...func(*iam.Options)) (*iam.GetOpenIDConnectProviderOutput, error)
	listOIDCProvidersFunc func(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput,
		optFns ...func(*iam.Options)) (*iam.ListOpenIDConnectProvidersOutput, error)
	tagOIDCProviderFunc func(ctx context.Context, params *iam.TagOpenIDConnectProviderInput,
		optFns ...func(*iam.Options)) (*iam.TagOpenIDConnectProviderOutput, error)
}

func (m *mockIAMClient) CreateOpenIDConnectProvider(ctx context.Context, params *iam.CreateOpenIDConnectProviderInput,
	optFns ...func(*iam.Options)) (*iam.CreateOpenIDConnectProviderOutput, error) {
	if m.createOIDCProviderFunc != nil {
		return m.createOIDCProviderFunc(ctx, params, optFns...)
	}
	return &iam.CreateOpenIDConnectProviderOutput{}, nil
}

func (m *mockIAMClient) GetOpenIDConnectProvider(ctx context.Context, params *iam.GetOpenIDConnectProviderInput,
	optFns ...func(*iam.Options)) (*iam.GetOpenIDConnectProviderOutput, error) {
	if m.getOIDCProviderFunc != nil {
		return m.getOIDCProviderFunc(ctx, params, optFns...)
	}
	return &iam.GetOpenIDConnectProviderOutput{}, nil
}

func (m *mockIAMClient) ListOpenIDConnectProviders(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput,
	optFns ...func(*iam.Options)) (*iam.ListOpenIDConnectProvidersOutput, error) {
	if m.listOIDCProvidersFunc != nil {
		return m.listOIDCProvidersFunc(ctx, params, optFns...)
	}
	return &iam.ListOpenIDConnectProvidersOutput{}, nil
}

func (m *mockIAMClient) TagOpenIDConnectProvider(ctx context.Context, params *iam.TagOpenIDConnectProviderInput,
	optFns ...func(*iam.Options)) (*iam.TagOpenIDConnectProviderOutput, error) {
	if m.tagOIDCProviderFunc != nil {
		return m.tagOIDCProviderFunc(ctx, params, optFns...)
	}
	return &iam.TagOpenIDConnectProviderOutput{}, nil
}

func TestValidateRequest(t *testing.T) {
	handler := NewHandler(&mockIAMClient{})

	tests := []struct {
		name        string
		req         OIDCProvisionerRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			req: OIDCProvisionerRequest{
				IssuerURL:  "https://example.com",
				Thumbprint: "abc123",
				ClusterID:  "test-cluster",
			},
			expectError: false,
		},
		{
			name: "missing issuer URL",
			req: OIDCProvisionerRequest{
				Thumbprint: "abc123",
				ClusterID:  "test-cluster",
			},
			expectError: true,
			errorMsg:    "issuer_url is required",
		},
		{
			name: "invalid issuer URL format",
			req: OIDCProvisionerRequest{
				IssuerURL:  "not-a-url",
				Thumbprint: "abc123",
				ClusterID:  "test-cluster",
			},
			expectError: true,
		},
		{
			name: "non-https issuer URL",
			req: OIDCProvisionerRequest{
				IssuerURL:  "http://example.com",
				Thumbprint: "abc123",
				ClusterID:  "test-cluster",
			},
			expectError: true,
			errorMsg:    "issuer_url must use https scheme",
		},
		{
			name: "missing thumbprint",
			req: OIDCProvisionerRequest{
				IssuerURL: "https://example.com",
				ClusterID: "test-cluster",
			},
			expectError: true,
			errorMsg:    "thumbprint is required",
		},
		{
			name: "missing cluster ID",
			req: OIDCProvisionerRequest{
				IssuerURL:  "https://example.com",
				Thumbprint: "abc123",
			},
			expectError: true,
			errorMsg:    "cluster_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateRequest(tt.req)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHandle_CreateNewProvider(t *testing.T) {
	ctx := context.Background()
	expectedARN := "arn:aws:iam::123456789012:oidc-provider/example.com"

	mock := &mockIAMClient{
		listOIDCProvidersFunc: func(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput,
			optFns ...func(*iam.Options)) (*iam.ListOpenIDConnectProvidersOutput, error) {
			return &iam.ListOpenIDConnectProvidersOutput{
				OpenIDConnectProviderList: []types.OpenIDConnectProviderListEntry{},
			}, nil
		},
		createOIDCProviderFunc: func(ctx context.Context, params *iam.CreateOpenIDConnectProviderInput,
			optFns ...func(*iam.Options)) (*iam.CreateOpenIDConnectProviderOutput, error) {
			assert.Equal(t, "https://example.com", *params.Url)
			assert.Equal(t, "abc123", params.ThumbprintList[0])
			assert.Contains(t, params.ClientIDList, "openshift")
			assert.Contains(t, params.ClientIDList, "sts.amazonaws.com")

			return &iam.CreateOpenIDConnectProviderOutput{
				OpenIDConnectProviderArn: aws.String(expectedARN),
			}, nil
		},
		tagOIDCProviderFunc: func(ctx context.Context, params *iam.TagOpenIDConnectProviderInput,
			optFns ...func(*iam.Options)) (*iam.TagOpenIDConnectProviderOutput, error) {
			assert.Equal(t, expectedARN, *params.OpenIDConnectProviderArn)
			assert.Len(t, params.Tags, 2)
			return &iam.TagOpenIDConnectProviderOutput{}, nil
		},
	}

	handler := NewHandler(mock)
	req := OIDCProvisionerRequest{
		IssuerURL:  "https://example.com",
		Thumbprint: "abc123",
		ClusterID:  "test-cluster",
	}

	resp, err := handler.Handle(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, expectedARN, resp.OIDCProviderARN)
	assert.Equal(t, statusCreated, resp.Status)
}

func TestHandle_ProviderAlreadyExists(t *testing.T) {
	ctx := context.Background()
	existingARN := "arn:aws:iam::123456789012:oidc-provider/example.com"

	mock := &mockIAMClient{
		listOIDCProvidersFunc: func(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput,
			optFns ...func(*iam.Options)) (*iam.ListOpenIDConnectProvidersOutput, error) {
			return &iam.ListOpenIDConnectProvidersOutput{
				OpenIDConnectProviderList: []types.OpenIDConnectProviderListEntry{
					{Arn: aws.String(existingARN)},
				},
			}, nil
		},
		getOIDCProviderFunc: func(ctx context.Context, params *iam.GetOpenIDConnectProviderInput,
			optFns ...func(*iam.Options)) (*iam.GetOpenIDConnectProviderOutput, error) {
			return &iam.GetOpenIDConnectProviderOutput{
				Url: aws.String("https://example.com"),
			}, nil
		},
		tagOIDCProviderFunc: func(ctx context.Context, params *iam.TagOpenIDConnectProviderInput,
			optFns ...func(*iam.Options)) (*iam.TagOpenIDConnectProviderOutput, error) {
			return &iam.TagOpenIDConnectProviderOutput{}, nil
		},
	}

	handler := NewHandler(mock)
	req := OIDCProvisionerRequest{
		IssuerURL:  "https://example.com",
		Thumbprint: "abc123",
		ClusterID:  "test-cluster",
	}

	resp, err := handler.Handle(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, existingARN, resp.OIDCProviderARN)
	assert.Equal(t, statusAlreadyExists, resp.Status)
}

func TestHandle_CreateWithCustomClientIDs(t *testing.T) {
	ctx := context.Background()
	expectedARN := "arn:aws:iam::123456789012:oidc-provider/example.com"
	customClientIDs := []string{"custom-client-1", "custom-client-2"}

	mock := &mockIAMClient{
		listOIDCProvidersFunc: func(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput,
			optFns ...func(*iam.Options)) (*iam.ListOpenIDConnectProvidersOutput, error) {
			return &iam.ListOpenIDConnectProvidersOutput{
				OpenIDConnectProviderList: []types.OpenIDConnectProviderListEntry{},
			}, nil
		},
		createOIDCProviderFunc: func(ctx context.Context, params *iam.CreateOpenIDConnectProviderInput,
			optFns ...func(*iam.Options)) (*iam.CreateOpenIDConnectProviderOutput, error) {
			assert.Equal(t, customClientIDs, params.ClientIDList)
			return &iam.CreateOpenIDConnectProviderOutput{
				OpenIDConnectProviderArn: aws.String(expectedARN),
			}, nil
		},
		tagOIDCProviderFunc: func(ctx context.Context, params *iam.TagOpenIDConnectProviderInput,
			optFns ...func(*iam.Options)) (*iam.TagOpenIDConnectProviderOutput, error) {
			return &iam.TagOpenIDConnectProviderOutput{}, nil
		},
	}

	handler := NewHandler(mock)
	req := OIDCProvisionerRequest{
		IssuerURL:  "https://example.com",
		Thumbprint: "abc123",
		ClusterID:  "test-cluster",
		ClientIDs:  customClientIDs,
	}

	resp, err := handler.Handle(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, expectedARN, resp.OIDCProviderARN)
}

func TestHandle_ErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		req       OIDCProvisionerRequest
		mockSetup func() *mockIAMClient
		expectErr bool
	}{
		{
			name: "list providers error",
			req: OIDCProvisionerRequest{
				IssuerURL:  "https://example.com",
				Thumbprint: "abc123",
				ClusterID:  "test-cluster",
			},
			mockSetup: func() *mockIAMClient {
				return &mockIAMClient{
					listOIDCProvidersFunc: func(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput,
						optFns ...func(*iam.Options)) (*iam.ListOpenIDConnectProvidersOutput, error) {
						return nil, errors.New("list error")
					},
				}
			},
			expectErr: true,
		},
		{
			name: "create provider error",
			req: OIDCProvisionerRequest{
				IssuerURL:  "https://example.com",
				Thumbprint: "abc123",
				ClusterID:  "test-cluster",
			},
			mockSetup: func() *mockIAMClient {
				return &mockIAMClient{
					listOIDCProvidersFunc: func(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput,
						optFns ...func(*iam.Options)) (*iam.ListOpenIDConnectProvidersOutput, error) {
						return &iam.ListOpenIDConnectProvidersOutput{}, nil
					},
					createOIDCProviderFunc: func(ctx context.Context, params *iam.CreateOpenIDConnectProviderInput,
						optFns ...func(*iam.Options)) (*iam.CreateOpenIDConnectProviderOutput, error) {
						return nil, errors.New("create error")
					},
				}
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.mockSetup()
			handler := NewHandler(mock)

			_, err := handler.Handle(context.Background(), tt.req)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckProviderExists_TrailingSlashHandling(t *testing.T) {
	ctx := context.Background()
	existingARN := "arn:aws:iam::123456789012:oidc-provider/example.com"

	mock := &mockIAMClient{
		listOIDCProvidersFunc: func(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput,
			optFns ...func(*iam.Options)) (*iam.ListOpenIDConnectProvidersOutput, error) {
			return &iam.ListOpenIDConnectProvidersOutput{
				OpenIDConnectProviderList: []types.OpenIDConnectProviderListEntry{
					{Arn: aws.String(existingARN)},
				},
			}, nil
		},
		getOIDCProviderFunc: func(ctx context.Context, params *iam.GetOpenIDConnectProviderInput,
			optFns ...func(*iam.Options)) (*iam.GetOpenIDConnectProviderOutput, error) {
			// Provider URL stored without trailing slash
			return &iam.GetOpenIDConnectProviderOutput{
				Url: aws.String("https://example.com"),
			}, nil
		},
	}

	handler := NewHandler(mock)

	// Test that we find the provider even if the request has a trailing slash
	arn, exists, err := handler.checkProviderExists(ctx, "https://example.com/")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, existingARN, arn)
}
