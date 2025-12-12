package config

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// CompanyKeyset holds company Stripe credentials from Parameter Store
type CompanyKeyset struct {
	SecretKey      string `json:"secret_key"`
	PublishableKey string `json:"publishable_key"`
	WebhookSecret  string `json:"webhook_secret"`
}

var (
	ssmClient     *ssm.Client
	ssmClientOnce sync.Once
)

func getSSMClient(ctx context.Context) (*ssm.Client, error) {
	var initErr error
	ssmClientOnce.Do(func() {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			initErr = fmt.Errorf("unable to load SDK config: %w", err)
			return
		}
		ssmClient = ssm.NewFromConfig(cfg)
	})
	return ssmClient, initErr
}

// GetParameter retrieves a parameter from SSM Parameter Store
func GetParameter(ctx context.Context, parameterName string, withDecryption bool) (string, error) {
	client, err := getSSMClient(ctx)
	if err != nil {
		return "", err
	}

	result, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(parameterName),
		WithDecryption: aws.Bool(withDecryption),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get parameter %s: %w", parameterName, err)
	}

	return *result.Parameter.Value, nil
}

// GetCompanyKeyset retrieves Stripe credentials for a company from Parameter Store
// Parameters are stored at: /payment-service/{env}/companies/{company_name}/stripe_keys
func GetCompanyKeyset(ctx context.Context, environment, companyName string) (*CompanyKeyset, error) {
	paramPath := fmt.Sprintf("/payment-service/%s/companies/%s/stripe_keys", environment, companyName)

	value, err := GetParameter(ctx, paramPath, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get company keyset for %s: %w", companyName, err)
	}

	var keyset CompanyKeyset
	if err := json.Unmarshal([]byte(value), &keyset); err != nil {
		return nil, fmt.Errorf("failed to parse company keyset: %w", err)
	}

	return &keyset, nil
}

// GetCompanySuccessURL builds the success redirect URL for a company
// URL format: https://{company_name}/api/payment/success
func GetCompanySuccessURL(companyName string) string {
	return fmt.Sprintf("https://%s/api/payment/success", companyName)
}
