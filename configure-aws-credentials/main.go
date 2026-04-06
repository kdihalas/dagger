// Configure AWS credentials for Dagger pipelines.
package main

import (
	"context"
	"dagger/aws-config/internal/dagger"
	"encoding/json"
	"fmt"
	"regexp"
)

var (
	validRoleArn     = regexp.MustCompile(`^arn:aws:iam::\d{12}:role/[\w+=,.@\-/]+$`)
	validSessionName = regexp.MustCompile(`^[\w+=,.@\-]{2,64}$`)
	validRegion      = regexp.MustCompile(`^[a-z]{2}(-[a-z]+-\d+)?$`)
)

const (
	awsCliImage    = "amazon/aws-cli:2.27.31"
	minDuration    = 900
	maxDuration    = 43200
	defaultSession = "dagger-session"
)

// AwsConfig configures AWS credentials for Dagger containers.
type AwsConfig struct {
	// +optional
	AccessKeyId *dagger.Secret
	// +optional
	SecretAccessKey *dagger.Secret
	// +optional
	SessionToken *dagger.Secret
	// +optional
	// +default="us-east-1"
	Region string
}

func New(
	// AWS access key ID.
	// +optional
	accessKeyId *dagger.Secret,
	// AWS secret access key.
	// +optional
	secretAccessKey *dagger.Secret,
	// AWS session token for temporary credentials.
	// +optional
	sessionToken *dagger.Secret,
	// AWS region.
	// +optional
	// +default="us-east-1"
	region string,
) *AwsConfig {
	return &AwsConfig{
		AccessKeyId:    accessKeyId,
		SecretAccessKey: secretAccessKey,
		SessionToken:   sessionToken,
		Region:         region,
	}
}

// WithCredentials applies AWS credentials to a container as environment variables.
func (a *AwsConfig) WithCredentials(
	// Container to configure with AWS credentials.
	ctr *dagger.Container,
) *dagger.Container {
	if a.AccessKeyId != nil {
		ctr = ctr.WithSecretVariable("AWS_ACCESS_KEY_ID", a.AccessKeyId)
	}
	if a.SecretAccessKey != nil {
		ctr = ctr.WithSecretVariable("AWS_SECRET_ACCESS_KEY", a.SecretAccessKey)
	}
	if a.SessionToken != nil {
		ctr = ctr.WithSecretVariable("AWS_SESSION_TOKEN", a.SessionToken)
	}
	ctr = ctr.WithEnvVariable("AWS_REGION", a.Region).
		WithEnvVariable("AWS_DEFAULT_REGION", a.Region)
	return ctr
}

func (a *AwsConfig) cliContainer() *dagger.Container {
	ctr := dag.Container().From(awsCliImage)
	if a.AccessKeyId != nil {
		ctr = ctr.WithSecretVariable("AWS_ACCESS_KEY_ID", a.AccessKeyId)
	}
	if a.SecretAccessKey != nil {
		ctr = ctr.WithSecretVariable("AWS_SECRET_ACCESS_KEY", a.SecretAccessKey)
	}
	if a.SessionToken != nil {
		ctr = ctr.WithSecretVariable("AWS_SESSION_TOKEN", a.SessionToken)
	}
	ctr = ctr.WithEnvVariable("AWS_REGION", a.Region).
		WithEnvVariable("AWS_DEFAULT_REGION", a.Region)
	return ctr
}

type stsResponse struct {
	Credentials struct {
		AccessKeyId     string `json:"AccessKeyId"`
		SecretAccessKey string `json:"SecretAccessKey"`
		SessionToken    string `json:"SessionToken"`
	} `json:"Credentials"`
}

func (a *AwsConfig) parseSTSResponse(ctx context.Context, out string) (*AwsConfig, error) {
	var resp stsResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		return nil, fmt.Errorf("parsing STS response: %w", err)
	}
	return &AwsConfig{
		AccessKeyId:    dag.SetSecret("aws-access-key-id", resp.Credentials.AccessKeyId),
		SecretAccessKey: dag.SetSecret("aws-secret-access-key", resp.Credentials.SecretAccessKey),
		SessionToken:   dag.SetSecret("aws-session-token", resp.Credentials.SessionToken),
		Region:         a.Region,
	}, nil
}

func validateDuration(duration int) error {
	if duration < minDuration || duration > maxDuration {
		return fmt.Errorf("duration must be between %d and %d seconds, got %d", minDuration, maxDuration, duration)
	}
	return nil
}

// AssumeRole calls STS AssumeRole and returns a new AwsConfig with temporary credentials.
func (a *AwsConfig) AssumeRole(
	ctx context.Context,
	// ARN of the IAM role to assume.
	// +required
	roleArn string,
	// Session name for the assumed role.
	// +optional
	// +default="dagger-session"
	sessionName string,
	// Duration in seconds (900-43200).
	// +optional
	// +default=3600
	duration int,
	// External ID for cross-account role assumption.
	// +optional
	externalId string,
	// Inline IAM policy JSON to scope down permissions.
	// +optional
	policy string,
) (*AwsConfig, error) {
	if !validRoleArn.MatchString(roleArn) {
		return nil, fmt.Errorf("invalid role ARN: %q", roleArn)
	}
	if sessionName != "" && !validSessionName.MatchString(sessionName) {
		return nil, fmt.Errorf("invalid session name: %q", sessionName)
	}
	if err := validateDuration(duration); err != nil {
		return nil, err
	}

	cmd := []string{"sts", "assume-role",
		"--role-arn", roleArn,
		"--role-session-name", sessionName,
		"--duration-seconds", fmt.Sprintf("%d", duration),
		"--output", "json",
	}
	if externalId != "" {
		cmd = append(cmd, "--external-id", externalId)
	}
	if policy != "" {
		cmd = append(cmd, "--policy", policy)
	}

	out, err := a.cliContainer().WithExec(cmd).Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("assuming role %s: %w", roleArn, err)
	}
	return a.parseSTSResponse(ctx, out)
}

// AssumeRoleWithWebIdentity calls STS AssumeRoleWithWebIdentity for OIDC-based auth
// and returns a new AwsConfig with temporary credentials.
func (a *AwsConfig) AssumeRoleWithWebIdentity(
	ctx context.Context,
	// ARN of the IAM role to assume.
	// +required
	roleArn string,
	// OIDC web identity token (e.g., GitHub Actions OIDC token).
	// +required
	webIdentityToken *dagger.Secret,
	// Session name for the assumed role.
	// +optional
	// +default="dagger-session"
	sessionName string,
	// Duration in seconds (900-43200).
	// +optional
	// +default=3600
	duration int,
	// Inline IAM policy JSON to scope down permissions.
	// +optional
	policy string,
) (*AwsConfig, error) {
	if !validRoleArn.MatchString(roleArn) {
		return nil, fmt.Errorf("invalid role ARN: %q", roleArn)
	}
	if sessionName != "" && !validSessionName.MatchString(sessionName) {
		return nil, fmt.Errorf("invalid session name: %q", sessionName)
	}
	if err := validateDuration(duration); err != nil {
		return nil, err
	}

	cmd := []string{"sts", "assume-role-with-web-identity",
		"--role-arn", roleArn,
		"--role-session-name", sessionName,
		"--duration-seconds", fmt.Sprintf("%d", duration),
		"--web-identity-token-file", "/tmp/web-identity-token",
		"--output", "json",
	}
	if policy != "" {
		cmd = append(cmd, "--policy", policy)
	}

	out, err := a.cliContainer().
		WithMountedSecret("/tmp/web-identity-token", webIdentityToken).
		WithExec(cmd).
		Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("assuming role with web identity %s: %w", roleArn, err)
	}
	return a.parseSTSResponse(ctx, out)
}
