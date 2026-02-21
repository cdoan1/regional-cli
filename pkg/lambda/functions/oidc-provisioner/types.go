package main

// OIDCProvisionerRequest represents the input to the OIDC provisioner Lambda
type OIDCProvisionerRequest struct {
	IssuerURL   string `json:"issuer_url"`
	Thumbprint  string `json:"thumbprint"`
	ClusterID   string `json:"cluster_id"`
	ClientIDs   []string `json:"client_ids,omitempty"`
}

// OIDCProvisionerResponse represents the output from the OIDC provisioner Lambda
type OIDCProvisionerResponse struct {
	OIDCProviderARN string `json:"oidc_provider_arn"`
	Status          string `json:"status"` // "created", "updated", "already_exists"
	Message         string `json:"message,omitempty"`
}

// OIDCProvisionerError represents an error response
type OIDCProvisionerError struct {
	ErrorType    string `json:"error_type"`
	ErrorMessage string `json:"error_message"`
}
