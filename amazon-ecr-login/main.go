// Log in to Amazon ECR and ECR Public registries.
package main

import (
	"context"
	"dagger/amazon-ecr-login/internal/dagger"
	"fmt"
	"regexp"
	"strings"
)

var validAccountId = regexp.MustCompile(`^\d{12}$`)

const (
	awsCliImage   = "amazon/aws-cli:2.27.31"
	ecrPublicHost = "public.ecr.aws"
)

// RegistryCredentials holds the login details for an ECR registry.
type RegistryCredentials struct {
	// The registry URL (e.g., 123456789012.dkr.ecr.us-east-1.amazonaws.com or public.ecr.aws).
	Registry string
	// Docker username (always "AWS" for ECR).
	Username string
	// Docker password (ECR authorization token).
	Password *dagger.Secret
}

// AmazonEcrLogin authenticates with Amazon ECR registries.
type AmazonEcrLogin struct {
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
	// AWS session token.
	// +optional
	sessionToken *dagger.Secret,
	// AWS region.
	// +optional
	// +default="us-east-1"
	region string,
) *AmazonEcrLogin {
	return &AmazonEcrLogin{
		AccessKeyId:    accessKeyId,
		SecretAccessKey: secretAccessKey,
		SessionToken:   sessionToken,
		Region:         region,
	}
}

func (e *AmazonEcrLogin) cliContainer() *dagger.Container {
	ctr := dag.Container().From(awsCliImage)
	if e.AccessKeyId != nil {
		ctr = ctr.WithSecretVariable("AWS_ACCESS_KEY_ID", e.AccessKeyId)
	}
	if e.SecretAccessKey != nil {
		ctr = ctr.WithSecretVariable("AWS_SECRET_ACCESS_KEY", e.SecretAccessKey)
	}
	if e.SessionToken != nil {
		ctr = ctr.WithSecretVariable("AWS_SESSION_TOKEN", e.SessionToken)
	}
	ctr = ctr.WithEnvVariable("AWS_REGION", e.Region).
		WithEnvVariable("AWS_DEFAULT_REGION", e.Region)
	return ctr
}

// Login authenticates with one or more private ECR registries and returns credentials.
func (e *AmazonEcrLogin) Login(
	ctx context.Context,
	// Comma-separated AWS account IDs. Defaults to the caller's account.
	// +optional
	registries string,
) ([]*RegistryCredentials, error) {
	cmd := []string{"ecr", "get-login-password", "--region", e.Region}
	out, err := e.cliContainer().WithExec(cmd).Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting ECR login password: %w", err)
	}
	password := strings.TrimSpace(out)

	var accountIds []string
	if registries != "" {
		for _, id := range strings.Split(registries, ",") {
			id = strings.TrimSpace(id)
			if !validAccountId.MatchString(id) {
				return nil, fmt.Errorf("invalid AWS account ID: %q", id)
			}
			accountIds = append(accountIds, id)
		}
	} else {
		// Get caller identity to determine default account
		identity, err := e.cliContainer().
			WithExec([]string{"sts", "get-caller-identity", "--query", "Account", "--output", "text"}).
			Stdout(ctx)
		if err != nil {
			return nil, fmt.Errorf("getting caller identity: %w", err)
		}
		accountIds = []string{strings.TrimSpace(identity)}
	}

	var creds []*RegistryCredentials
	for _, accountId := range accountIds {
		registry := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", accountId, e.Region)
		creds = append(creds, &RegistryCredentials{
			Registry: registry,
			Username: "AWS",
			Password: dag.SetSecret(fmt.Sprintf("ecr-password-%s", accountId), password),
		})
	}
	return creds, nil
}

// LoginPublic authenticates with ECR Public (public.ecr.aws) and returns credentials.
func (e *AmazonEcrLogin) LoginPublic(
	ctx context.Context,
) (*RegistryCredentials, error) {
	// ECR Public always uses us-east-1
	cmd := []string{"ecr-public", "get-login-password", "--region", "us-east-1"}
	out, err := e.cliContainer().
		WithEnvVariable("AWS_REGION", "us-east-1").
		WithEnvVariable("AWS_DEFAULT_REGION", "us-east-1").
		WithExec(cmd).Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting ECR Public login password: %w", err)
	}
	password := strings.TrimSpace(out)

	return &RegistryCredentials{
		Registry: ecrPublicHost,
		Username: "AWS",
		Password: dag.SetSecret("ecr-public-password", password),
	}, nil
}

// WithRegistryAuth applies ECR registry authentication to a container.
func (e *AmazonEcrLogin) WithRegistryAuth(
	ctx context.Context,
	// Container to authenticate with ECR.
	ctr *dagger.Container,
	// Comma-separated AWS account IDs. Defaults to the caller's account.
	// +optional
	registries string,
) (*dagger.Container, error) {
	creds, err := e.Login(ctx, registries)
	if err != nil {
		return nil, err
	}
	for _, c := range creds {
		ctr = ctr.WithRegistryAuth(c.Registry, c.Username, c.Password)
	}
	return ctr, nil
}

// WithPublicRegistryAuth applies ECR Public registry authentication to a container.
func (e *AmazonEcrLogin) WithPublicRegistryAuth(
	ctx context.Context,
	// Container to authenticate with ECR Public.
	ctr *dagger.Container,
) (*dagger.Container, error) {
	cred, err := e.LoginPublic(ctx)
	if err != nil {
		return nil, err
	}
	return ctr.WithRegistryAuth(cred.Registry, cred.Username, cred.Password), nil
}
