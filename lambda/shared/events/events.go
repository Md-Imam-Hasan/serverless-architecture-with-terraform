package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

var (
	eventBridgeClient     *eventbridge.Client
	eventBridgeClientOnce sync.Once
)

func getEventBridgeClient(ctx context.Context) (*eventbridge.Client, error) {
	var initErr error
	eventBridgeClientOnce.Do(func() {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			initErr = fmt.Errorf("unable to load SDK config: %w", err)
			return
		}
		eventBridgeClient = eventbridge.NewFromConfig(cfg)
	})
	return eventBridgeClient, initErr
}

// PaymentLinkCreatedEvent represents the event data for payment link created
type PaymentLinkCreatedEvent struct {
	PaymentID         string `json:"paymentId"`
	OrderNumber       string `json:"orderNumber"`
	PaymentLink       string `json:"paymentLink"`
	PaymentLinkID     string `json:"paymentLinkId"`
	ProviderPaymentID string `json:"providerPaymentId"`
	CompanyName       string `json:"companyName"`
	Amount            string `json:"amount"`
	Currency          string `json:"currency"`
	Status            string `json:"status"`
	CreatedAt         string `json:"createdAt"`
}

// PaymentLinkFailedEvent represents the event data for payment link generation failure
type PaymentLinkFailedEvent struct {
	PaymentID     string `json:"paymentId"`
	OrderNumber   string `json:"orderNumber"`
	CompanyName   string `json:"companyName"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
	FailureReason string `json:"failureReason"`
	FailureCount  int    `json:"failureCount"`
	FailedAt      string `json:"failedAt"`
}

// PaymentStatusEvent represents a generic payment status event
type PaymentStatusEvent struct {
	PaymentID         string `json:"paymentId"`
	OrderNumber       string `json:"orderNumber"`
	CompanyName       string `json:"companyName"`
	Amount            string `json:"amount"`
	Currency          string `json:"currency"`
	Status            string `json:"status"`
	ProviderPaymentID string `json:"providerPaymentId,omitempty"`
	UpdatedAt         string `json:"updatedAt"`
	PaidAt            string `json:"paidAt,omitempty"`
	AmountReceived    string `json:"amountReceived,omitempty"`
	FailureMessage    string `json:"failureMessage,omitempty"`
	RefundID          string `json:"refundId,omitempty"`
	RefundedAt        string `json:"refundedAt,omitempty"`
	RefundStatus      string `json:"refundStatus,omitempty"`
}

// PublishEvent publishes an event to EventBridge
func PublishEvent(ctx context.Context, eventBusName, source, detailType string, detail interface{}) error {
	client, err := getEventBridgeClient(ctx)
	if err != nil {
		return err
	}

	detailJSON, err := json.Marshal(detail)
	if err != nil {
		return fmt.Errorf("failed to marshal event detail: %w", err)
	}

	input := &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				Source:       aws.String(source),
				DetailType:   aws.String(detailType),
				Detail:       aws.String(string(detailJSON)),
				EventBusName: aws.String(eventBusName),
			},
		},
	}

	result, err := client.PutEvents(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	if result.FailedEntryCount > 0 {
		return fmt.Errorf("failed to publish %d events", result.FailedEntryCount)
	}

	return nil
}
