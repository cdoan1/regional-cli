# rosactl - ROSA Regional HCP CLI

`rosactl` is the command-line interface for the ROSA Regional HCP (Hosted Control Plane) platform. It enables customers to provision and manage HyperShift clusters with AWS IAM authentication.

## Overview

Phase 1 of `rosactl` focuses on deploying the OIDC provisioner Lambda function to customer AWS accounts. This Lambda function creates OIDC providers needed for cluster authentication.

## Features

- **AWS Credential Validation**: Validates AWS credentials and region configuration
- **Platform API Connectivity Check**: Tests connectivity to the ROSA Regional platform API
- **OIDC Lambda Deployment**: Deploys Lambda function for creating OIDC providers in customer accounts
- **IAM Role Management**: Automatically creates and configures Lambda execution roles
- **CloudWatch Logs Integration**: Sets up log groups with 90-day retention
- **Idempotent Operations**: Safe to re-run commands without side effects

## Installation

### Prerequisites

- Go 1.23 or later
- AWS CLI configured with valid credentials
- AWS account with appropriate permissions

### Build from Source

```bash
# Clone the repository
git clone https://github.com/openshift-online/regional-cli.git
cd regional-cli

# Build the CLI
make build

# Install to $GOPATH/bin
make install
```

The binary will be available at `bin/rosactl`.

## Usage

### Global Flags

All commands support these global flags:

- `--profile <name>`: AWS credential profile to use
- `--region <region>`: AWS region (e.g., us-east-1)
- `--verbose`, `-v`: Enable verbose logging
- `--platform-api-url <url>`: Platform API endpoint URL

### Commands

#### `rosactl init`

Validates your AWS environment and configuration.

**What it checks:**
- AWS credentials are valid
- AWS region is configured and supported
- Platform API is reachable (if URL provided)

**Example:**

```bash
# Basic validation
rosactl init --region us-east-1

# With verbose output
rosactl init --region us-east-1 --verbose

# With Platform API check
rosactl init --region us-east-1 --platform-api-url https://api.rosa.example.com

# Using a specific AWS profile
rosactl init --profile my-aws-profile --region us-west-2
```

**Output:**

```
✓ AWS credentials valid
✓ Platform API reachable

Validation complete. Your environment is configured correctly.
```

#### `rosactl setup-account`

Deploys the OIDC provisioner Lambda function to your AWS account.

**What it does:**
- Creates Lambda execution IAM role (if it doesn't exist)
- Builds the Lambda deployment package
- Deploys the OIDC provisioner Lambda function
- Configures CloudWatch Log Group with 90-day retention
- Optionally adds resource policy for CLM service role invocation

**Example:**

```bash
# Basic deployment
rosactl setup-account --region us-east-1

# With custom function name
rosactl setup-account --region us-east-1 --function-name my-oidc-provisioner

# With CLM resource policy
rosactl setup-account \
  --region us-east-1 \
  --clm-service-role-arn arn:aws:iam::123456789012:role/clm-service-role \
  --source-account-id 123456789012

# With verbose output
rosactl setup-account --region us-east-1 --verbose
```

**Flags:**

- `--function-name`: Lambda function name (default: `rosa-oidc-provisioner`)
- `--execution-role-name`: Lambda execution role name (default: `rosa-oidc-provisioner-execution`)
- `--clm-service-role-arn`: CLM service role ARN for resource-based policy
- `--source-account-id`: AWS account ID for resource-based policy

**Output:**

```
Deploying OIDC provisioner Lambda function...
✓ Lambda function created: rosa-oidc-provisioner
✓ IAM execution role created
✓ CloudWatch Log Group created

Setup complete. Lambda function deployed: arn:aws:lambda:us-east-1:123456789012:function:rosa-oidc-provisioner
Your AWS account is now configured for ROSA cluster provisioning.
```

## Architecture

### Components

1. **OIDC Provisioner Lambda**: AWS Lambda function that creates OIDC providers for cluster authentication
2. **Lambda Execution Role**: IAM role with minimal permissions for OIDC provider management
3. **CloudWatch Logs**: Log group for Lambda execution logs with 90-day retention
4. **Resource Policy**: Optional policy allowing CLM service role to invoke the Lambda

### AWS Permissions Required

To run `rosactl setup-account`, your AWS credentials need:

**IAM Permissions:**
- `iam:CreateRole`
- `iam:GetRole`
- `iam:PutRolePolicy`

**Lambda Permissions:**
- `lambda:CreateFunction`
- `lambda:GetFunction`
- `lambda:UpdateFunctionCode`
- `lambda:UpdateFunctionConfiguration`
- `lambda:AddPermission`
- `lambda:TagResource`

**CloudWatch Logs Permissions:**
- `logs:CreateLogGroup`
- `logs:DescribeLogGroups`
- `logs:PutRetentionPolicy`
- `logs:TagLogGroup`

### Lambda Function Details

The deployed OIDC provisioner Lambda has the following configuration:

- **Runtime**: `provided.al2023` (Go custom runtime)
- **Memory**: 128 MB
- **Timeout**: 60 seconds
- **Architecture**: x86_64
- **Handler**: `bootstrap`

## Development

### Project Structure

```
regional-cli/
├── cmd/
│   └── rosactl/          # CLI entry point
├── internal/
│   ├── aws/              # AWS client wrappers
│   ├── cli/              # CLI commands
│   └── validator/        # Validation logic
├── pkg/
│   └── lambda/
│       ├── deployer/     # Lambda deployment orchestrator
│       └── functions/
│           └── oidc-provisioner/  # OIDC Lambda function
└── Makefile              # Build automation
```

### Building

```bash
# Build CLI binary
make build

# Build Lambda function for Linux/AMD64
make build-lambda

# Run tests
make test

# Run tests with coverage
make test-coverage

# Generate HTML coverage report
make coverage-html

# Run linter
make lint

# Clean build artifacts
make clean
```

### Testing

The project has comprehensive unit test coverage:

- **Lambda Handler**: 80.3% coverage
- **Lambda Deployer**: 78.7% coverage
- **AWS Validators**: 96.3% coverage
- **AWS Clients**: 92.3% coverage

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run tests for specific package
go test ./pkg/lambda/functions/oidc-provisioner/... -v
```

## Troubleshooting

### Common Issues

#### "AWS credentials validation failed"

**Cause**: AWS credentials are not configured or invalid.

**Solution**:
```bash
# Configure AWS credentials
aws configure

# Or use environment variables
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export AWS_REGION=us-east-1

# Or specify a profile
rosactl init --profile my-profile --region us-east-1
```

#### "AWS region 'xyz' is not supported"

**Cause**: The specified region is not in the supported regions list.

**Supported regions**:
- US: `us-east-1`, `us-east-2`, `us-west-1`, `us-west-2`
- EU: `eu-west-1`, `eu-west-2`, `eu-west-3`, `eu-central-1`, `eu-north-1`
- Asia Pacific: `ap-southeast-1`, `ap-southeast-2`, `ap-northeast-1`, `ap-northeast-2`, `ap-south-1`
- Other: `sa-east-1`, `ca-central-1`

**Solution**: Use one of the supported regions.

#### "Platform API validation failed"

**Cause**: Unable to connect to the Platform API endpoint.

**Solution**:
- Verify the API URL is correct
- Check network connectivity
- Ensure firewall rules allow outbound HTTPS
- Try without `--platform-api-url` flag to skip this check

#### "Deployment failed: compilation failed"

**Cause**: Go build environment issue or missing dependencies.

**Solution**:
```bash
# Update dependencies
go mod tidy

# Ensure Go 1.23+ is installed
go version

# Try building manually
make build-lambda
```

#### "AccessDenied" errors during setup-account

**Cause**: AWS credentials lack required permissions.

**Solution**: Ensure your AWS user/role has the permissions listed in the "AWS Permissions Required" section above.

## Supported Regions

rosactl currently supports the following AWS regions:

- **US East**: us-east-1, us-east-2
- **US West**: us-west-1, us-west-2
- **Europe**: eu-west-1, eu-west-2, eu-west-3, eu-central-1, eu-north-1
- **Asia Pacific**: ap-southeast-1, ap-southeast-2, ap-northeast-1, ap-northeast-2, ap-south-1
- **South America**: sa-east-1
- **Canada**: ca-central-1

## Version

Current version: **0.1.0** (Phase 1)

## Contributing

This project is under active development. Contribution guidelines will be published in future releases.

## License

[License information to be added]

## Support

For issues and questions:
- GitHub Issues: https://github.com/openshift-online/regional-cli/issues
- Documentation: [To be added]

## Roadmap

### Phase 1 (Current) ✓
- AWS credential validation
- Platform API connectivity check
- OIDC provisioner Lambda deployment

### Phase 2 (Planned)
- Cluster creation workflow
- Network verification Lambda
- Preflight checker Lambda
- Platform API integration

### Phase 3 (Future)
- Cluster management commands
- Upgrade workflows
- Multi-cluster operations
- Enhanced monitoring and logging
