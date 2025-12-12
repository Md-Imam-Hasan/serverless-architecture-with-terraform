package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	ebtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentlink"
	"github.com/stripe/stripe-go/v76/price"
	"github.com/stripe/stripe-go/v76/product"
)

// ============================================================================
// TYPES AND CONSTANTS
// ============================================================================

type PaymentStatus string

const (
	PaymentStatusLinkGenerated        PaymentStatus = "link_generated"
	PaymentStatusLinkGenerationFailed PaymentStatus = "link_generation_failed"
	PaymentStatusCanceled             PaymentStatus = "canceled"
)

type EventType string

const (
	EventTypePaymentLinkCreated          EventType = "payment_link.created"
	EventTypePaymentLinkGenerationFailed EventType = "payment_link.generation_failed"
)

// StepFunctionPayload represents the input from Step Functions
type StepFunctionPayload struct {
	PaymentID   string  `json:"paymentId"`
	OrderNumber string  `json:"orderNumber"`
	CompanyName string  `json:"companyName"`
	Amount      int     `json:"amount"` // In cents
	Currency    string  `json:"currency"`
	SourceKey   string  `json:"sourceKey"`
	TripID      *string `json:"tripId,omitempty"`
	ScheduledAt *string `json:"scheduledAt,omitempty"`
	RequestID   *string `json:"requestId,omitempty"`
}

// PaymentRecord represents a payment in DynamoDB
type PaymentRecord struct {
	PK                string  `dynamodbav:"PK"`
	SK                string  `dynamodbav:"SK"`
	PaymentID         string  `dynamodbav:"paymentId"`
	OrderNumber       string  `dynamodbav:"orderNumber"`
	CompanyName       string  `dynamodbav:"companyName"`
	Status            string  `dynamodbav:"status"`
	Amount            int     `dynamodbav:"amount"`
	Currency          string  `dynamodbav:"currency"`
	SourceKey         string  `dynamodbav:"sourceKey"`
	CreatedAt         string  `dynamodbav:"createdAt"`
	PaymentLink       *string `dynamodbav:"paymentLink,omitempty"`
	PaymentLinkID     *string `dynamodbav:"paymentLinkId,omitempty"`
	ProviderPaymentID *string `dynamodbav:"providerPaymentId,omitempty"`
	FailureCount      *int    `dynamodbav:"failureCount,omitempty"`
}

// StepFunctionResponse represents the response to Step Functions
type StepFunctionResponse struct {
	StatusCode int    `json:"statusCode"`
	PaymentID  string `json:"paymentId"`
	Status     string `json:"status,omitempty"`
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
	CreatedAt  string `json:"createdAt,omitempty"`
}

type CompanyKeyset struct {
	SecretKey      string `json:"secret_key"`
	PublishableKey string `json:"publishable_key"`
	WebhookSecret  string `json:"webhook_secret"`
}

type CompanyConfig struct {
	BaseURL    string `json:"base_url"`
	SuccessAPI string `json:"success_api"`
}

// ============================================================================
// GLOBAL VARIABLES
// ============================================================================

var (
	dynamoClient      *dynamodb.Client
	eventBridgeClient *eventbridge.Client
	ssmClient         *ssm.Client
	tableName         string
	eventBusName      string
	environment       string
	initOnce          sync.Once
)

func initClients() {
	initOnce.Do(func() {
		tableName = os.Getenv("PAYMENTS_TABLE_NAME")
		eventBusName = os.Getenv("EVENT_BUS_NAME")
		environment = os.Getenv("ENVIRONMENT")

		if eventBusName == "" {
			eventBusName = "default"
		}

		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatalf("unable to load SDK config: %v", err)
		}

		dynamoClient = dynamodb.NewFromConfig(cfg)
		eventBridgeClient = eventbridge.NewFromConfig(cfg)
		ssmClient = ssm.NewFromConfig(cfg)
	})
}

// ============================================================================
// VALIDATION FUNCTIONS
// ============================================================================

func validateRequiredFields(payload *StepFunctionPayload) error {
	if payload.PaymentID == "" {
		return errors.New("missing required field: paymentId")
	}
	if payload.OrderNumber == "" {
		return errors.New("missing required field: orderNumber")
	}
	if payload.CompanyName == "" {
		return errors.New("missing required field: companyName")
	}
	if payload.Amount == 0 {
		return errors.New("missing required field: amount")
	}
	if payload.Currency == "" {
		return errors.New("missing required field: currency")
	}
	return nil
}

func getPayment(ctx context.Context, paymentID string) (*PaymentRecord, error) {
	result, err := dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PAYMENT#%s", paymentID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("payment %s not found in database", paymentID)
	}

	var payment PaymentRecord
	if err := attributevalue.UnmarshalMap(result.Item, &payment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment: %w", err)
	}

	return &payment, nil
}

// ============================================================================
// STRIPE LINK GENERATION
// ============================================================================

func getCompanyKeyset(ctx context.Context, companyName string) (*CompanyKeyset, error) {
	paramPath := fmt.Sprintf("/payment-service/%s/companies/%s/stripe_keys", environment, companyName)

	result, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(paramPath),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get company keyset: %w", err)
	}

	var keyset CompanyKeyset
	if err := json.Unmarshal([]byte(*result.Parameter.Value), &keyset); err != nil {
		return nil, fmt.Errorf("failed to parse company keyset: %w", err)
	}

	return &keyset, nil
}

func getCompanyConfig(ctx context.Context, companyName string) (*CompanyConfig, error) {
	paramPath := fmt.Sprintf("/payment-service/%s/companies/%s/config", environment, companyName)

	result, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(paramPath),
		WithDecryption: aws.Bool(false),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get company config: %w", err)
	}

	var cfg CompanyConfig
	if err := json.Unmarshal([]byte(*result.Parameter.Value), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse company config: %w", err)
	}

	return &cfg, nil
}

type stripeResult struct {
	paymentLink       string
	paymentLinkID     string
	providerPaymentID string
	createdAt         string
}

func generateStripeLink(ctx context.Context, payload *StepFunctionPayload, traceID string) (*stripeResult, error) {
	// Get Stripe keys
	keyset, err := getCompanyKeyset(ctx, payload.CompanyName)
	if err != nil {
		return nil, err
	}

	// Get redirect URL from config
	var redirectURL *string
	cfg, err := getCompanyConfig(ctx, payload.CompanyName)
	if err == nil {
		url := cfg.BaseURL + cfg.SuccessAPI
		redirectURL = &url
	}

	stripe.Key = keyset.SecretKey

	// Create product
	description := fmt.Sprintf("Payment for order %s", payload.OrderNumber)
	productParams := &stripe.ProductParams{
		Name: stripe.String(description),
	}
	productParams.AddMetadata("paymentId", payload.PaymentID)
	productParams.AddMetadata("companyName", payload.CompanyName)
	productParams.AddMetadata("sourceKey", payload.SourceKey)
	productParams.AddMetadata("orderNumber", payload.OrderNumber)

	prod, err := product.New(productParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Create price
	priceParams := &stripe.PriceParams{
		Product:    stripe.String(prod.ID),
		UnitAmount: stripe.Int64(int64(payload.Amount)),
		Currency:   stripe.String(payload.Currency),
	}

	pr, err := price.New(priceParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create price: %w", err)
	}

	// Create payment link
	linkParams := &stripe.PaymentLinkParams{
		LineItems: []*stripe.PaymentLinkLineItemParams{
			{
				Price:    stripe.String(pr.ID),
				Quantity: stripe.Int64(1),
			},
		},
	}

	if redirectURL != nil && *redirectURL != "" {
		linkParams.AfterCompletion = &stripe.PaymentLinkAfterCompletionParams{
			Type: stripe.String("redirect"),
			Redirect: &stripe.PaymentLinkAfterCompletionRedirectParams{
				URL: stripe.String(*redirectURL),
			},
		}
	}

	linkParams.AddMetadata("paymentId", payload.PaymentID)
	linkParams.AddMetadata("companyName", payload.CompanyName)
	linkParams.AddMetadata("sourceKey", payload.SourceKey)
	linkParams.AddMetadata("orderNumber", payload.OrderNumber)

	linkParams.PaymentIntentData = &stripe.PaymentLinkPaymentIntentDataParams{}
	linkParams.PaymentIntentData.AddMetadata("paymentId", payload.PaymentID)
	linkParams.PaymentIntentData.AddMetadata("companyName", payload.CompanyName)
	linkParams.PaymentIntentData.AddMetadata("sourceKey", payload.SourceKey)
	linkParams.PaymentIntentData.AddMetadata("orderNumber", payload.OrderNumber)

	link, err := paymentlink.New(linkParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment link: %w", err)
	}

	return &stripeResult{
		paymentLink:       link.URL,
		paymentLinkID:     link.ID,
		providerPaymentID: prod.ID,
		createdAt:         time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// ============================================================================
// DATABASE OPERATIONS
// ============================================================================

func updatePaymentWithLink(ctx context.Context, paymentID string, result *stripeResult) error {
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := dynamoClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PAYMENT#%s", paymentID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET #status = :status, #paymentLink = :paymentLink, #paymentLinkId = :paymentLinkId, #providerPaymentId = :providerPaymentId, #updatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#status":            "status",
			"#paymentLink":       "paymentLink",
			"#paymentLinkId":     "paymentLinkId",
			"#providerPaymentId": "providerPaymentId",
			"#updatedAt":         "updatedAt",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":            &types.AttributeValueMemberS{Value: string(PaymentStatusLinkGenerated)},
			":paymentLink":       &types.AttributeValueMemberS{Value: result.paymentLink},
			":paymentLinkId":     &types.AttributeValueMemberS{Value: result.paymentLinkID},
			":providerPaymentId": &types.AttributeValueMemberS{Value: result.providerPaymentID},
			":updatedAt":         &types.AttributeValueMemberS{Value: now},
		},
	})

	return err
}

func updatePaymentWithFailure(ctx context.Context, paymentID, errorMessage string, existingPayment *PaymentRecord) error {
	now := time.Now().UTC().Format(time.RFC3339)

	failureCount := 1
	if existingPayment.FailureCount != nil {
		failureCount = *existingPayment.FailureCount + 1
	}

	_, err := dynamoClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PAYMENT#%s", paymentID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET #status = :status, #failureReason = :failureReason, #failureCount = :failureCount, #lastFailedAt = :lastFailedAt, #updatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#status":        "status",
			"#failureReason": "failureReason",
			"#failureCount":  "failureCount",
			"#lastFailedAt":  "lastFailedAt",
			"#updatedAt":     "updatedAt",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":        &types.AttributeValueMemberS{Value: string(PaymentStatusLinkGenerationFailed)},
			":failureReason": &types.AttributeValueMemberS{Value: errorMessage},
			":failureCount":  &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", failureCount)},
			":lastFailedAt":  &types.AttributeValueMemberS{Value: now},
			":updatedAt":     &types.AttributeValueMemberS{Value: now},
		},
	})

	return err
}

// ============================================================================
// EVENTBRIDGE PUBLISHING
// ============================================================================

func publishPaymentLinkCreatedEvent(ctx context.Context, payload *StepFunctionPayload, result *stripeResult) error {
	eventDetail := map[string]interface{}{
		"paymentId":         payload.PaymentID,
		"orderNumber":       payload.OrderNumber,
		"paymentLink":       result.paymentLink,
		"paymentLinkId":     result.paymentLinkID,
		"providerPaymentId": result.providerPaymentID,
		"companyName":       payload.CompanyName,
		"amount":            fmt.Sprintf("%d", payload.Amount),
		"currency":          payload.Currency,
		"status":            string(PaymentStatusLinkGenerated),
		"createdAt":         result.createdAt,
	}

	detailJSON, _ := json.Marshal(eventDetail)

	_, err := eventBridgeClient.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []ebtypes.PutEventsRequestEntry{
			{
				Source:       aws.String("payment-service"),
				DetailType:   aws.String(string(EventTypePaymentLinkCreated)),
				Detail:       aws.String(string(detailJSON)),
				EventBusName: aws.String(eventBusName),
			},
		},
	})

	return err
}

func publishPaymentLinkFailedEvent(ctx context.Context, payload *StepFunctionPayload, errorMessage string, failureCount int) error {
	now := time.Now().UTC().Format(time.RFC3339)

	eventDetail := map[string]interface{}{
		"paymentId":     payload.PaymentID,
		"orderNumber":   payload.OrderNumber,
		"companyName":   payload.CompanyName,
		"amount":        fmt.Sprintf("%d", payload.Amount),
		"currency":      payload.Currency,
		"status":        string(PaymentStatusLinkGenerationFailed),
		"failureReason": errorMessage,
		"failureCount":  failureCount,
		"failedAt":      now,
	}

	detailJSON, _ := json.Marshal(eventDetail)

	_, err := eventBridgeClient.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []ebtypes.PutEventsRequestEntry{
			{
				Source:       aws.String("payment-service"),
				DetailType:   aws.String(string(EventTypePaymentLinkGenerationFailed)),
				Detail:       aws.String(string(detailJSON)),
				EventBusName: aws.String(eventBusName),
			},
		},
	})

	return err
}

// ============================================================================
// MAIN HANDLER
// ============================================================================

func handler(ctx context.Context, event json.RawMessage) (StepFunctionResponse, error) {
	initClients()

	traceID := fmt.Sprintf("step-%d", time.Now().UnixNano())
	log.Printf(`{"message": "Step handler invoked", "event": %s, "traceId": "%s"}`, string(event), traceID)

	// Parse payload - handle both direct and wrapped payloads
	var payload StepFunctionPayload

	// Try to parse as wrapped payload first
	var wrappedPayload struct {
		Payload StepFunctionPayload `json:"Payload"`
	}
	if err := json.Unmarshal(event, &wrappedPayload); err == nil && wrappedPayload.Payload.PaymentID != "" {
		payload = wrappedPayload.Payload
	} else {
		// Parse as direct payload
		if err := json.Unmarshal(event, &payload); err != nil {
			log.Printf(`{"error": "Failed to parse payload", "details": "%s", "traceId": "%s"}`, err.Error(), traceID)
			return StepFunctionResponse{}, fmt.Errorf("failed to parse payload: %w", err)
		}
	}

	// Phase 1: Validate payload
	if err := validateRequiredFields(&payload); err != nil {
		log.Printf(`{"error": "Validation error", "details": "%s", "traceId": "%s"}`, err.Error(), traceID)
		return StepFunctionResponse{}, err
	}

	// Phase 2: Check if payment exists
	existingPayment, err := getPayment(ctx, payload.PaymentID)
	if err != nil {
		log.Printf(`{"error": "Payment not found", "details": "%s", "traceId": "%s"}`, err.Error(), traceID)
		return StepFunctionResponse{}, err
	}

	// Phase 2.5: Check if payment was canceled
	if existingPayment.Status == string(PaymentStatusCanceled) {
		log.Printf(`{"message": "Payment was canceled, skipping link generation", "paymentId": "%s"}`, payload.PaymentID)
		return StepFunctionResponse{
			StatusCode: 200,
			PaymentID:  payload.PaymentID,
			Status:     string(PaymentStatusCanceled),
			Message:    "Payment was canceled, link generation skipped",
		}, nil
	}

	// Phase 3: Check idempotency - if payment link already exists
	if existingPayment.PaymentLink != nil && *existingPayment.PaymentLink != "" {
		log.Printf(`{"message": "Payment link already exists (idempotent)", "paymentId": "%s"}`, payload.PaymentID)

		// Still publish event for consistency
		_ = publishPaymentLinkCreatedEvent(ctx, &payload, &stripeResult{
			paymentLink:       *existingPayment.PaymentLink,
			paymentLinkID:     *existingPayment.PaymentLinkID,
			providerPaymentID: *existingPayment.ProviderPaymentID,
			createdAt:         existingPayment.CreatedAt,
		})

		return StepFunctionResponse{
			StatusCode: 200,
			PaymentID:  payload.PaymentID,
			CreatedAt:  existingPayment.CreatedAt,
			Message:    "Payment link already exists (idempotent)",
		}, nil
	}

	// Phase 4: Generate Stripe payment link
	stripeResult, stripeErr := generateStripeLink(ctx, &payload, traceID)

	// Phase 5: Update database based on result
	if stripeErr != nil {
		log.Printf(`{"error": "Stripe link generation failed", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
			stripeErr.Error(), payload.PaymentID, traceID)

		// Update payment with failure
		_ = updatePaymentWithFailure(ctx, payload.PaymentID, stripeErr.Error(), existingPayment)

		// Publish failure event
		failureCount := 1
		if existingPayment.FailureCount != nil {
			failureCount = *existingPayment.FailureCount + 1
		}
		_ = publishPaymentLinkFailedEvent(ctx, &payload, stripeErr.Error(), failureCount)

		// Return success response (Step Function completes successfully)
		return StepFunctionResponse{
			StatusCode: 200,
			PaymentID:  payload.PaymentID,
			Status:     string(PaymentStatusLinkGenerationFailed),
			Message:    "Payment saved but Stripe link generation failed. Retry mechanism will handle this.",
			Error:      stripeErr.Error(),
		}, nil
	}

	// Update payment with link
	if err := updatePaymentWithLink(ctx, payload.PaymentID, stripeResult); err != nil {
		log.Printf(`{"error": "Failed to update payment", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
			err.Error(), payload.PaymentID, traceID)
	}

	// Phase 6: Publish success event
	if err := publishPaymentLinkCreatedEvent(ctx, &payload, stripeResult); err != nil {
		log.Printf(`{"error": "Failed to publish event", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
			err.Error(), payload.PaymentID, traceID)
	}

	log.Printf(`{"message": "Successfully processed scheduled payment", "paymentId": "%s", "paymentLink": "%s", "traceId": "%s"}`,
		payload.PaymentID, stripeResult.paymentLink, traceID)

	return StepFunctionResponse{
		StatusCode: 200,
		PaymentID:  payload.PaymentID,
		CreatedAt:  stripeResult.createdAt,
		Message:    "Payment link generated successfully",
	}, nil
}

func main() {
	lambda.Start(handler)
}
