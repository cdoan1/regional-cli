package deployer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateLambdaExecutionRoleTrustPolicy(t *testing.T) {
	policyStr, err := GenerateLambdaExecutionRoleTrustPolicy()
	require.NoError(t, err)
	assert.NotEmpty(t, policyStr)

	// Verify it's valid JSON
	var policy PolicyDocument
	err = json.Unmarshal([]byte(policyStr), &policy)
	require.NoError(t, err)

	// Verify policy structure
	assert.Equal(t, "2012-10-17", policy.Version)
	assert.Len(t, policy.Statement, 1)

	stmt := policy.Statement[0]
	assert.Equal(t, "Allow", stmt.Effect)
	assert.Equal(t, "sts:AssumeRole", stmt.Action)

	// Verify principal
	principal, ok := stmt.Principal["Service"]
	assert.True(t, ok)
	assert.Equal(t, "lambda.amazonaws.com", principal)
}

func TestGenerateOIDCProvisionerPermissionsPolicy(t *testing.T) {
	policyStr, err := GenerateOIDCProvisionerPermissionsPolicy()
	require.NoError(t, err)
	assert.NotEmpty(t, policyStr)

	// Verify it's valid JSON
	var policy PolicyDocument
	err = json.Unmarshal([]byte(policyStr), &policy)
	require.NoError(t, err)

	// Verify policy structure
	assert.Equal(t, "2012-10-17", policy.Version)
	assert.Len(t, policy.Statement, 2)

	// Verify IAM permissions
	iamStmt := policy.Statement[0]
	assert.Equal(t, "Allow", iamStmt.Effect)

	actions, ok := iamStmt.Action.([]interface{})
	assert.True(t, ok)
	assert.Contains(t, toString(actions), "iam:CreateOpenIDConnectProvider")
	assert.Contains(t, toString(actions), "iam:GetOpenIDConnectProvider")
	assert.Contains(t, toString(actions), "iam:ListOpenIDConnectProviders")
	assert.Contains(t, toString(actions), "iam:TagOpenIDConnectProvider")

	// Verify CloudWatch Logs permissions
	logsStmt := policy.Statement[1]
	assert.Equal(t, "Allow", logsStmt.Effect)

	logsActions, ok := logsStmt.Action.([]interface{})
	assert.True(t, ok)
	assert.Contains(t, toString(logsActions), "logs:CreateLogGroup")
	assert.Contains(t, toString(logsActions), "logs:CreateLogStream")
	assert.Contains(t, toString(logsActions), "logs:PutLogEvents")
}

func TestGenerateLambdaResourcePolicy(t *testing.T) {
	tests := []struct {
		name              string
		clmRoleARN        string
		sourceAccountID   string
		expectError       bool
		expectedErrorMsg  string
	}{
		{
			name:            "valid policy",
			clmRoleARN:      "arn:aws:iam::123456789012:role/clm-service-role",
			sourceAccountID: "123456789012",
			expectError:     false,
		},
		{
			name:             "missing CLM role ARN",
			clmRoleARN:       "",
			sourceAccountID:  "123456789012",
			expectError:      true,
			expectedErrorMsg: "CLM service role ARN is required",
		},
		{
			name:             "missing source account ID",
			clmRoleARN:       "arn:aws:iam::123456789012:role/clm-service-role",
			sourceAccountID:  "",
			expectError:      true,
			expectedErrorMsg: "source account ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policyStr, err := GenerateLambdaResourcePolicy(tt.clmRoleARN, tt.sourceAccountID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				assert.Empty(t, policyStr)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, policyStr)

				// Verify it's valid JSON
				var policy PolicyDocument
				err = json.Unmarshal([]byte(policyStr), &policy)
				require.NoError(t, err)

				// Verify policy structure
				assert.Equal(t, "2012-10-17", policy.Version)
				assert.Len(t, policy.Statement, 1)

				stmt := policy.Statement[0]
				assert.Equal(t, "Allow", stmt.Effect)
				assert.Equal(t, "lambda:InvokeFunction", stmt.Action)

				// Verify principal
				principal, ok := stmt.Principal["AWS"]
				assert.True(t, ok)
				assert.Equal(t, tt.clmRoleARN, principal)

				// Verify condition
				assert.NotNil(t, stmt.Condition)
				stringEquals, ok := stmt.Condition["StringEquals"].(map[string]interface{})
				assert.True(t, ok)
				sourceAccount, ok := stringEquals["aws:SourceAccount"]
				assert.True(t, ok)
				assert.Equal(t, tt.sourceAccountID, sourceAccount)
			}
		})
	}
}

func TestPolicyJSONFormat(t *testing.T) {
	// Verify all policies produce valid, well-formed JSON
	policies := []struct {
		name     string
		generate func() (string, error)
	}{
		{"Trust Policy", func() (string, error) { return GenerateLambdaExecutionRoleTrustPolicy() }},
		{"Permissions Policy", func() (string, error) { return GenerateOIDCProvisionerPermissionsPolicy() }},
		{"Resource Policy", func() (string, error) {
			return GenerateLambdaResourcePolicy("arn:aws:iam::123456789012:role/test", "123456789012")
		}},
	}

	for _, p := range policies {
		t.Run(p.name, func(t *testing.T) {
			policyStr, err := p.generate()
			require.NoError(t, err)

			// Verify it can be unmarshaled and remarshaled
			var doc PolicyDocument
			err = json.Unmarshal([]byte(policyStr), &doc)
			require.NoError(t, err)

			// Verify remarshaling produces valid JSON
			_, err = json.Marshal(doc)
			require.NoError(t, err)
		})
	}
}

// Helper function to convert []interface{} to []string for assertions
func toString(slice []interface{}) []string {
	result := make([]string, len(slice))
	for i, v := range slice {
		result[i] = v.(string)
	}
	return result
}
