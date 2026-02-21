package deployer

import (
	"encoding/json"
	"fmt"
)

// PolicyDocument represents an AWS IAM policy document
type PolicyDocument struct {
	Version   string      `json:"Version"`
	Statement []Statement `json:"Statement"`
}

// Statement represents a policy statement
type Statement struct {
	Effect    string                 `json:"Effect"`
	Principal map[string]interface{} `json:"Principal,omitempty"`
	Action    interface{}            `json:"Action"` // Can be string or []string
	Resource  interface{}            `json:"Resource,omitempty"`
	Condition map[string]interface{} `json:"Condition,omitempty"`
}

// GenerateLambdaExecutionRoleTrustPolicy generates the trust policy for Lambda execution role
func GenerateLambdaExecutionRoleTrustPolicy() (string, error) {
	policy := PolicyDocument{
		Version: "2012-10-17",
		Statement: []Statement{
			{
				Effect: "Allow",
				Principal: map[string]interface{}{
					"Service": "lambda.amazonaws.com",
				},
				Action: "sts:AssumeRole",
			},
		},
	}

	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal trust policy: %w", err)
	}

	return string(policyJSON), nil
}

// GenerateOIDCProvisionerPermissionsPolicy generates the permissions policy for OIDC provisioner Lambda
func GenerateOIDCProvisionerPermissionsPolicy() (string, error) {
	policy := PolicyDocument{
		Version: "2012-10-17",
		Statement: []Statement{
			{
				Effect: "Allow",
				Action: []string{
					"iam:CreateOpenIDConnectProvider",
					"iam:GetOpenIDConnectProvider",
					"iam:ListOpenIDConnectProviders",
					"iam:TagOpenIDConnectProvider",
				},
				Resource: "*",
			},
			{
				Effect: "Allow",
				Action: []string{
					"logs:CreateLogGroup",
					"logs:CreateLogStream",
					"logs:PutLogEvents",
				},
				Resource: "arn:aws:logs:*:*:*",
			},
		},
	}

	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal permissions policy: %w", err)
	}

	return string(policyJSON), nil
}

// GenerateLambdaResourcePolicy generates a resource-based policy allowing CLM service role to invoke the Lambda
func GenerateLambdaResourcePolicy(clmServiceRoleARN string, sourceAccountID string) (string, error) {
	if clmServiceRoleARN == "" {
		return "", fmt.Errorf("CLM service role ARN is required")
	}

	if sourceAccountID == "" {
		return "", fmt.Errorf("source account ID is required")
	}

	policy := PolicyDocument{
		Version: "2012-10-17",
		Statement: []Statement{
			{
				Effect: "Allow",
				Principal: map[string]interface{}{
					"AWS": clmServiceRoleARN,
				},
				Action:   "lambda:InvokeFunction",
				Resource: "*",
				Condition: map[string]interface{}{
					"StringEquals": map[string]string{
						"aws:SourceAccount": sourceAccountID,
					},
				},
			},
		},
	}

	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal resource policy: %w", err)
	}

	return string(policyJSON), nil
}
