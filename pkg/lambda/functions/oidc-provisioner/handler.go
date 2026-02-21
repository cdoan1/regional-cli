package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

const (
	statusCreated       = "created"
	statusAlreadyExists = "already_exists"
	tagComponentKey     = "rosa:component"
	tagComponentValue   = "oidc-provider"
	tagClusterKey       = "rosa:cluster-id"
)

// IAMAPI defines the IAM operations needed by the handler
type IAMAPI interface {
	CreateOpenIDConnectProvider(ctx context.Context, params *iam.CreateOpenIDConnectProviderInput,
		optFns ...func(*iam.Options)) (*iam.CreateOpenIDConnectProviderOutput, error)
	GetOpenIDConnectProvider(ctx context.Context, params *iam.GetOpenIDConnectProviderInput,
		optFns ...func(*iam.Options)) (*iam.GetOpenIDConnectProviderOutput, error)
	ListOpenIDConnectProviders(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput,
		optFns ...func(*iam.Options)) (*iam.ListOpenIDConnectProvidersOutput, error)
	TagOpenIDConnectProvider(ctx context.Context, params *iam.TagOpenIDConnectProviderInput,
		optFns ...func(*iam.Options)) (*iam.TagOpenIDConnectProviderOutput, error)
}

// Handler handles OIDC provider creation requests
type Handler struct {
	iamClient IAMAPI
}

// NewHandler creates a new OIDC provisioner handler
func NewHandler(iamClient IAMAPI) *Handler {
	return &Handler{
		iamClient: iamClient,
	}
}

// Handle processes the OIDC provisioner request
func (h *Handler) Handle(ctx context.Context, req OIDCProvisionerRequest) (*OIDCProvisionerResponse, error) {
	// Validate request
	if err := h.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Normalize issuer URL (remove trailing slash)
	issuerURL := strings.TrimSuffix(req.IssuerURL, "/")

	// Check if provider already exists
	providerARN, exists, err := h.checkProviderExists(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to check if provider exists: %w", err)
	}

	if exists {
		// Provider already exists, ensure tags are set
		if err := h.tagProvider(ctx, providerARN, req.ClusterID); err != nil {
			return nil, fmt.Errorf("failed to tag existing provider: %w", err)
		}

		return &OIDCProvisionerResponse{
			OIDCProviderARN: providerARN,
			Status:          statusAlreadyExists,
			Message:         "OIDC provider already exists",
		}, nil
	}

	// Create new OIDC provider
	providerARN, err = h.createProvider(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	// Tag the newly created provider
	if err := h.tagProvider(ctx, providerARN, req.ClusterID); err != nil {
		// Don't fail if tagging fails (provider is already created)
		// Just log the error (Lambda logs will capture it)
		fmt.Printf("Warning: failed to tag provider: %v\n", err)
	}

	return &OIDCProvisionerResponse{
		OIDCProviderARN: providerARN,
		Status:          statusCreated,
		Message:         "OIDC provider created successfully",
	}, nil
}

// validateRequest validates the input request
func (h *Handler) validateRequest(req OIDCProvisionerRequest) error {
	if req.IssuerURL == "" {
		return errors.New("issuer_url is required")
	}

	// Validate URL format
	parsedURL, err := url.Parse(req.IssuerURL)
	if err != nil {
		return fmt.Errorf("invalid issuer_url: %w", err)
	}

	if parsedURL.Scheme != "https" {
		return errors.New("issuer_url must use https scheme")
	}

	if parsedURL.Host == "" {
		return errors.New("issuer_url must have a valid host")
	}

	if req.Thumbprint == "" {
		return errors.New("thumbprint is required")
	}

	if req.ClusterID == "" {
		return errors.New("cluster_id is required")
	}

	return nil
}

// checkProviderExists checks if an OIDC provider with the given issuer URL already exists
func (h *Handler) checkProviderExists(ctx context.Context, issuerURL string) (string, bool, error) {
	// Normalize issuer URL (remove trailing slash)
	normalizedIssuerURL := strings.TrimSuffix(issuerURL, "/")

	output, err := h.iamClient.ListOpenIDConnectProviders(ctx, &iam.ListOpenIDConnectProvidersInput{})
	if err != nil {
		return "", false, err
	}

	// Check each provider to see if it matches our issuer URL
	for _, provider := range output.OpenIDConnectProviderList {
		// GetOpenIDConnectProvider returns the URL without the "arn:" prefix
		getOutput, err := h.iamClient.GetOpenIDConnectProvider(ctx, &iam.GetOpenIDConnectProviderInput{
			OpenIDConnectProviderArn: provider.Arn,
		})
		if err != nil {
			// If we can't get details, skip this provider
			continue
		}

		if getOutput.Url != nil && strings.TrimSuffix(*getOutput.Url, "/") == normalizedIssuerURL {
			return *provider.Arn, true, nil
		}
	}

	return "", false, nil
}

// createProvider creates a new OIDC provider
func (h *Handler) createProvider(ctx context.Context, req OIDCProvisionerRequest) (string, error) {
	input := &iam.CreateOpenIDConnectProviderInput{
		Url:            aws.String(strings.TrimSuffix(req.IssuerURL, "/")),
		ThumbprintList: []string{req.Thumbprint},
	}

	// Add client IDs if provided
	if len(req.ClientIDs) > 0 {
		input.ClientIDList = req.ClientIDs
	} else {
		// Use default client IDs for ROSA
		input.ClientIDList = []string{
			"openshift",
			"sts.amazonaws.com",
		}
	}

	output, err := h.iamClient.CreateOpenIDConnectProvider(ctx, input)
	if err != nil {
		return "", err
	}

	return *output.OpenIDConnectProviderArn, nil
}

// tagProvider adds tags to the OIDC provider
func (h *Handler) tagProvider(ctx context.Context, providerARN, clusterID string) error {
	tags := []types.Tag{
		{
			Key:   aws.String(tagComponentKey),
			Value: aws.String(tagComponentValue),
		},
	}

	if clusterID != "" {
		tags = append(tags, types.Tag{
			Key:   aws.String(tagClusterKey),
			Value: aws.String(clusterID),
		})
	}

	_, err := h.iamClient.TagOpenIDConnectProvider(ctx, &iam.TagOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: aws.String(providerARN),
		Tags:                     tags,
	})

	return err
}
