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
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentlink"
	"github.com/stripe/stripe-go/v76/price"
	"github.com/stripe/stripe-go/v76/product"

	sharedconfig "github.com/payment-service/shared/config"
	"github.com/payment-service/shared/enums"
	sharedevents "github.com/payment-service/shared/events"
	"github.com/payment-service/shared/response"
)

type StepFunctionPayload struct {
	PaymentID   string  `json:"paymentId"`
	OrderNumber string  `json:"orderNumber"`
	CompanyName string  `json:"companyName"`
	Amount      int     `json:"amount"`
	Currency    string  `json:"currency"`
	SourceKey   string  `json:"sourceKey"`
	TripID      *string `json:"tripId,omitempty"`
	ScheduledAt *string `json:"scheduledAt,omitempty"`
	RequestID   *string `json:"requestId,omitempty"`
}

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

type stripeResult struct {
	paymentLink       string
	paymentLinkID     string
	providerPaymentID string
	createdAt         string
}

var (
	dynamoClient *dynamodb.Client
	tableName    string
	eventBusName string
	environment  string
	initOnce     sync.Once
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
	})
}

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
			":status":            &types.AttributeValueMemberS{Value: enums.PaymentStatusLinkGenerated.String()},
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
			":status":        &types.AttributeValueMemberS{Value: enums.PaymentStatusLinkGenerationFailed.String()},
			":failureReason": &types.AttributeValueMemberS{Value: errorMessage},
			":failureCount":  &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", failureCount)},
			":lastFailedAt":  &types.AttributeValueMemberS{Value: now},
			":updatedAt":     &types.AttributeValueMemberS{Value: now},
		},
	})
	return err
}

func generateStripeLink(ctx context.Context, payload *StepFunctionPayload) (*stripeResult, error) {
	keyset, err := sharedconfig.GetCompanyKeyset(ctx, environment, payload.CompanyName)
	if err != nil {
		return nil, err
	}
	redirectURL := sharedconfig.GetCompanySuccessURL(payload.CompanyName)
	stripe.Key = keyset.SecretKey

	description := fmt.Sprintf("Payment for order %s", payload.OrderNumber)
	productParams := &stripe.ProductParams{Name: stripe.String(description)}
	productParams.AddMetadata("paymentId", payload.PaymentID)
	productParams.AddMetadata("companyName", payload.CompanyName)
	productParams.AddMetadata("sourceKey", payload.SourceKey)
	productParams.AddMetadata("orderNumber", payload.OrderNumber)

	prod, err := product.New(productParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	priceParams := &stripe.PriceParams{
		Product:    stripe.String(prod.ID),
		UnitAmount: stripe.Int64(int64(payload.Amount)),
		Currency:   stripe.String(payload.Currency),
	}
	pr, err := price.New(priceParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create price: %w", err)
	}

	linkParams := &stripe.PaymentLinkParams{
		LineItems: []*stripe.PaymentLinkLineItemParams{
			{Price: stripe.String(pr.ID), Quantity: stripe.Int64(1)},
		},
	}
	linkParams.AfterCompletion = &stripe.PaymentLinkAfterCompletionParams{
		Type:     stripe.String("redirect"),
		Redirect: &stripe.PaymentLinkAfterCompletionRedirectParams{URL: stripe.String(redirectURL)},
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

func publishPaymentLinkCreatedEvent(ctx context.Context, payload *StepFunctionPayload, result *stripeResult) error {
	event := sharedevents.PaymentLinkCreatedEvent{
		PaymentID:         payload.PaymentID,
		OrderNumber:       payload.OrderNumber,
		PaymentLink:       result.paymentLink,
		PaymentLinkID:     result.paymentLinkID,
		ProviderPaymentID: result.providerPaymentID,
		CompanyName:       payload.CompanyName,
		Amount:            fmt.Sprintf("%d", payload.Amount),
		Currency:          payload.Currency,
		Status:            enums.PaymentStatusLinkGenerated.String(),
		CreatedAt:         result.createdAt,
	}
	return sharedevents.PublishEvent(ctx, eventBusName, "payment-service", enums.EventTypePaymentLinkCreated.String(), event)
}

func publishPaymentLinkFailedEvent(ctx context.Context, payload *StepFunctionPayload, errorMessage string, failureCount int) error {
	now := time.Now().UTC().Format(time.RFC3339)
	event := sharedevents.PaymentLinkFailedEvent{
		PaymentID:     payload.PaymentID,
		OrderNumber:   payload.OrderNumber,
		CompanyName:   payload.CompanyName,
		Amount:        fmt.Sprintf("%d", payload.Amount),
		Currency:      payload.Currency,
		Status:        enums.PaymentStatusLinkGenerationFailed.String(),
		FailureReason: errorMessage,
		FailureCount:  failureCount,
		FailedAt:      now,
	}
	return sharedevents.PublishEvent(ctx, eventBusName, "payment-service", enums.EventTypePaymentLinkGenerationFailed.String(), event)
}

func handler(ctx context.Context, event json.RawMessage) (response.StepFunctionResponse, error) {
	initClients()
	traceID := fmt.Sprintf("step-%d", time.Now().UnixNano())
	log.Printf(`{"message": "Step handler invoked", "event": %s, "traceId": "%s"}`, string(event), traceID)

	var payload StepFunctionPayload
	var wrappedPayload struct {
		Payload StepFunctionPayload `json:"Payload"`
	}
	if err := json.Unmarshal(event, &wrappedPayload); err == nil && wrappedPayload.Payload.PaymentID != "" {
		payload = wrappedPayload.Payload
	} else {
		if err := json.Unmarshal(event, &payload); err != nil {
			log.Printf(`{"error": "Failed to parse payload", "details": "%s", "traceId": "%s"}`, err.Error(), traceID)
			return response.StepFunctionResponse{}, fmt.Errorf("failed to parse payload: %w", err)
		}
	}

	if err := validateRequiredFields(&payload); err != nil {
		log.Printf(`{"error": "Validation error", "details": "%s", "traceId": "%s"}`, err.Error(), traceID)
		return response.StepFunctionResponse{}, err
	}

	existingPayment, err := getPayment(ctx, payload.PaymentID)
	if err != nil {
		log.Printf(`{"error": "Payment not found", "details": "%s", "traceId": "%s"}`, err.Error(), traceID)
		return response.StepFunctionResponse{}, err
	}

	if existingPayment.Status == enums.PaymentStatusCanceled.String() {
		log.Printf(`{"message": "Payment was canceled, skipping link generation", "paymentId": "%s"}`, payload.PaymentID)
		return response.StepFunctionResponse{
			StatusCode: 200,
			PaymentID:  payload.PaymentID,
			Status:     enums.PaymentStatusCanceled.String(),
			Message:    "Payment was canceled, link generation skipped",
		}, nil
	}

	if existingPayment.PaymentLink != nil && *existingPayment.PaymentLink != "" {
		log.Printf(`{"message": "Payment link already exists (idempotent)", "paymentId": "%s"}`, payload.PaymentID)
		_ = publishPaymentLinkCreatedEvent(ctx, &payload, &stripeResult{
			paymentLink:       *existingPayment.PaymentLink,
			paymentLinkID:     *existingPayment.PaymentLinkID,
			providerPaymentID: *existingPayment.ProviderPaymentID,
			createdAt:         existingPayment.CreatedAt,
		})
		return response.StepFunctionResponse{
			StatusCode: 200,
			PaymentID:  payload.PaymentID,
			CreatedAt:  existingPayment.CreatedAt,
			Message:    "Payment link already exists (idempotent)",
		}, nil
	}

	stripeResult, stripeErr := generateStripeLink(ctx, &payload)
	if stripeErr != nil {
		log.Printf(`{"error": "Stripe link generation failed", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
			stripeErr.Error(), payload.PaymentID, traceID)
		_ = updatePaymentWithFailure(ctx, payload.PaymentID, stripeErr.Error(), existingPayment)
		failureCount := 1
		if existingPayment.FailureCount != nil {
			failureCount = *existingPayment.FailureCount + 1
		}
		_ = publishPaymentLinkFailedEvent(ctx, &payload, stripeErr.Error(), failureCount)
		return response.StepFunctionResponse{
			StatusCode: 200,
			PaymentID:  payload.PaymentID,
			Status:     enums.PaymentStatusLinkGenerationFailed.String(),
			Message:    "Payment saved but Stripe link generation failed. Retry mechanism will handle this.",
			Error:      stripeErr.Error(),
		}, nil
	}

	if err := updatePaymentWithLink(ctx, payload.PaymentID, stripeResult); err != nil {
		log.Printf(`{"error": "Failed to update payment", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
			err.Error(), payload.PaymentID, traceID)
	}

	if err := publishPaymentLinkCreatedEvent(ctx, &payload, stripeResult); err != nil {
		log.Printf(`{"error": "Failed to publish event", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
			err.Error(), payload.PaymentID, traceID)
	}

	log.Printf(`{"message": "Successfully processed scheduled payment", "paymentId": "%s", "paymentLink": "%s", "traceId": "%s"}`,
		payload.PaymentID, stripeResult.paymentLink, traceID)

	return response.StepFunctionResponse{
		StatusCode: 200,
		PaymentID:  payload.PaymentID,
		CreatedAt:  stripeResult.createdAt,
		Message:    "Payment link generated successfully",
	}, nil
}

func main() {
	lambda.Start(handler)
}
