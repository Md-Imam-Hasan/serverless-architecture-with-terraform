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

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	ebtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentlink"
	"github.com/stripe/stripe-go/v76/webhook"

	sharedconfig "github.com/payment-service/shared/config"
	"github.com/payment-service/shared/enums"
)

// ============================================================================
// TYPES AND CONSTANTS
// ============================================================================

// Using enums from shared module

type PaymentRecord struct {
	PK                string  `dynamodbav:"PK"`
	SK                string  `dynamodbav:"SK"`
	PaymentID         string  `dynamodbav:"paymentId"`
	OrderNumber       string  `dynamodbav:"orderNumber"`
	CompanyName       string  `dynamodbav:"companyName"`
	Status            string  `dynamodbav:"status"`
	Amount            int     `dynamodbav:"amount"`
	Currency          string  `dynamodbav:"currency"`
	PaymentLinkID     *string `dynamodbav:"paymentLinkId,omitempty"`
	ProviderPaymentID *string `dynamodbav:"providerPaymentId,omitempty"`
}

// Using CompanyKeyset from shared/config module

type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type SuccessResponse struct {
	Received  bool        `json:"received"`
	EventType string      `json:"eventType"`
	Result    interface{} `json:"result,omitempty"`
	Handled   *bool       `json:"handled,omitempty"`
}

// ============================================================================
// GLOBAL VARIABLES
// ============================================================================

var (
	dynamoClient      *dynamodb.Client
	eventBridgeClient *eventbridge.Client
	tableName         string
	eventBusName      string
	environment       string
	initOnce          sync.Once
)

var defaultHeaders = map[string]string{
	"Content-Type":                     "application/json",
	"Access-Control-Allow-Origin":      "*",
	"Access-Control-Allow-Headers":     "Content-Type,Authorization,X-Api-Key,Stripe-Signature",
	"Access-Control-Allow-Methods":     "OPTIONS,POST",
	"Access-Control-Allow-Credentials": "false",
}

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
	})
}

// ============================================================================
// RESPONSE HELPERS
// ============================================================================

func errorResponse(statusCode int, errCode, message string, details map[string]interface{}) events.APIGatewayProxyResponse {
	body, _ := json.Marshal(ErrorResponse{
		Error:   errCode,
		Message: message,
		Details: details,
	})
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    defaultHeaders,
		Body:       string(body),
	}
}

func successResponse(statusCode int, data interface{}) events.APIGatewayProxyResponse {
	body, _ := json.Marshal(data)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    defaultHeaders,
		Body:       string(body),
	}
}

// ============================================================================
// VALIDATION FUNCTIONS
// ============================================================================

func getCompanyKeyset(ctx context.Context, companyName string) (*sharedconfig.CompanyKeyset, error) {
	return sharedconfig.GetCompanyKeyset(ctx, environment, companyName)
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
		return nil, errors.New("payment not found")
	}

	var payment PaymentRecord
	if err := attributevalue.UnmarshalMap(result.Item, &payment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment: %w", err)
	}

	return &payment, nil
}

// ============================================================================
// DATABASE OPERATIONS
// ============================================================================

func updatePaymentStatus(ctx context.Context, paymentID, status string, additionalFields map[string]interface{}, allowedStatuses []string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	updateExpr := "SET #status = :status, #updatedAt = :updatedAt"
	exprAttrNames := map[string]string{
		"#status":    "status",
		"#updatedAt": "updatedAt",
	}
	exprAttrValues := map[string]types.AttributeValue{
		":status":    &types.AttributeValueMemberS{Value: status},
		":updatedAt": &types.AttributeValueMemberS{Value: now},
	}

	// Add additional fields
	i := 0
	for key, value := range additionalFields {
		if key == "status" || key == "updatedAt" {
			continue
		}
		placeholder := fmt.Sprintf("#field%d", i)
		valuePlaceholder := fmt.Sprintf(":val%d", i)
		updateExpr += fmt.Sprintf(", %s = %s", placeholder, valuePlaceholder)
		exprAttrNames[placeholder] = key

		av, err := attributevalue.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal field %s: %w", key, err)
		}
		exprAttrValues[valuePlaceholder] = av
		i++
	}

	// Build condition expression if allowed statuses provided
	var conditionExpr *string
	if len(allowedStatuses) > 0 {
		condition := "#status IN ("
		for j, allowedStatus := range allowedStatuses {
			placeholder := fmt.Sprintf(":allowedStatus%d", j)
			if j > 0 {
				condition += ", "
			}
			condition += placeholder
			exprAttrValues[placeholder] = &types.AttributeValueMemberS{Value: allowedStatus}
		}
		condition += ")"
		conditionExpr = aws.String(condition)
	}

	_, err := dynamoClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PAYMENT#%s", paymentID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression:          aws.String(updateExpr),
		ConditionExpression:       conditionExpr,
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
	})

	return err
}

// ============================================================================
// EVENTBRIDGE PUBLISHING
// ============================================================================

func publishPaymentStatusEvent(ctx context.Context, payment *PaymentRecord, eventType string, additionalFields map[string]interface{}) error {
	eventDetail := map[string]interface{}{
		"paymentId":   payment.PaymentID,
		"orderNumber": payment.OrderNumber,
		"companyName": payment.CompanyName,
		"amount":      fmt.Sprintf("%d", payment.Amount),
		"currency":    payment.Currency,
		"status":      payment.Status,
		"updatedAt":   time.Now().UTC().Format(time.RFC3339),
	}

	for k, v := range additionalFields {
		eventDetail[k] = v
	}

	detailJSON, _ := json.Marshal(eventDetail)

	result, err := eventBridgeClient.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []ebtypes.PutEventsRequestEntry{
			{
				Source:       aws.String("payment-service"),
				DetailType:   aws.String(eventType),
				Detail:       aws.String(string(detailJSON)),
				EventBusName: aws.String(eventBusName),
			},
		},
	})
	if err != nil {
		return err
	}

	if result.FailedEntryCount > 0 {
		return fmt.Errorf("failed to publish %d events", result.FailedEntryCount)
	}

	return nil
}

// ============================================================================
// EVENT HANDLERS
// ============================================================================

func handlePaymentIntentSucceeded(ctx context.Context, paymentIntent map[string]interface{}, traceID string) (map[string]string, error) {
	metadata, _ := paymentIntent["metadata"].(map[string]interface{})
	paymentID, _ := metadata["paymentId"].(string)
	companyName, _ := metadata["companyName"].(string)

	if paymentID == "" {
		return nil, errors.New("missing paymentId in payment_intent metadata")
	}

	// Get current payment
	payment, err := getPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	// Idempotency check
	if payment.Status == string(enums.PaymentStatusPaid) {
		return map[string]string{"paymentId": paymentID, "status": string(enums.PaymentStatusPaid)}, nil
	}

	// Allowed source statuses
	allowedStatuses := []string{string(enums.PaymentStatusProcessing), string(enums.PaymentStatusLinkGenerated)}

	amountReceived, _ := paymentIntent["amount_received"].(float64)
	providerPaymentID, _ := paymentIntent["id"].(string)

	additionalFields := map[string]interface{}{
		"paidAt":            time.Now().UTC().Format(time.RFC3339),
		"providerPaymentId": providerPaymentID,
		"amountReceived":    amountReceived / 100,
	}

	if err := updatePaymentStatus(ctx, paymentID, string(enums.PaymentStatusPaid), additionalFields, allowedStatuses); err != nil {
		log.Printf(`{"error": "Failed to update payment status", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
			err.Error(), paymentID, traceID)
	}

	// Deactivate payment link
	if payment.PaymentLinkID != nil && companyName != "" {
		keyset, err := getCompanyKeyset(ctx, companyName)
		if err == nil {
			stripe.Key = keyset.SecretKey
			_, err = paymentlink.Update(*payment.PaymentLinkID, &stripe.PaymentLinkParams{
				Active: stripe.Bool(false),
			})
			if err != nil {
				log.Printf(`{"warning": "Failed to deactivate payment link", "paymentId": "%s", "error": "%s"}`,
					paymentID, err.Error())
			}
		}
	}

	// Refresh payment and publish event
	payment, _ = getPayment(ctx, paymentID)
	_ = publishPaymentStatusEvent(ctx, payment, string(enums.EventTypePaymentIntentSucceeded), additionalFields)

	return map[string]string{"paymentId": paymentID, "status": string(enums.PaymentStatusPaid)}, nil
}

func handlePaymentIntentFailed(ctx context.Context, paymentIntent map[string]interface{}, traceID string) (map[string]string, error) {
	metadata, _ := paymentIntent["metadata"].(map[string]interface{})
	paymentID, _ := metadata["paymentId"].(string)

	if paymentID == "" {
		return nil, errors.New("missing paymentId in payment_intent metadata")
	}

	// Get current payment
	payment, err := getPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	// Idempotency check
	if payment.Status == string(enums.PaymentStatusFailed) {
		return map[string]string{"paymentId": paymentID, "status": string(enums.PaymentStatusFailed)}, nil
	}

	// Protected statuses
	if payment.Status == string(enums.PaymentStatusPaid) || payment.Status == string(enums.PaymentStatusRefunded) {
		return map[string]string{"paymentId": paymentID, "status": payment.Status}, nil
	}

	allowedStatuses := []string{string(enums.PaymentStatusProcessing), string(enums.PaymentStatusLinkGenerated), "created"}

	lastPaymentError, _ := paymentIntent["last_payment_error"].(map[string]interface{})
	failureMessage, _ := lastPaymentError["message"].(string)
	if failureMessage == "" {
		failureMessage = "Payment failed"
	}
	providerPaymentID, _ := paymentIntent["id"].(string)

	additionalFields := map[string]interface{}{
		"providerPaymentId": providerPaymentID,
		"failureMessage":    failureMessage,
	}

	if err := updatePaymentStatus(ctx, paymentID, string(enums.PaymentStatusFailed), additionalFields, allowedStatuses); err != nil {
		log.Printf(`{"error": "Failed to update payment status", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
			err.Error(), paymentID, traceID)
	}

	payment, _ = getPayment(ctx, paymentID)
	_ = publishPaymentStatusEvent(ctx, payment, string(enums.EventTypePaymentIntentFailed), additionalFields)

	return map[string]string{"paymentId": paymentID, "status": string(enums.PaymentStatusFailed)}, nil
}

func handlePaymentIntentCanceled(ctx context.Context, paymentIntent map[string]interface{}, traceID string) (map[string]string, error) {
	metadata, _ := paymentIntent["metadata"].(map[string]interface{})
	paymentID, _ := metadata["paymentId"].(string)

	if paymentID == "" {
		return nil, errors.New("missing paymentId in payment_intent metadata")
	}

	payment, err := getPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	// Idempotency check
	if payment.Status == string(enums.PaymentStatusCanceled) {
		return map[string]string{"paymentId": paymentID, "status": string(enums.PaymentStatusCanceled)}, nil
	}

	// Protected statuses
	if payment.Status == string(enums.PaymentStatusPaid) || payment.Status == string(enums.PaymentStatusRefunded) {
		return map[string]string{"paymentId": paymentID, "status": payment.Status}, nil
	}

	allowedStatuses := []string{"created", string(enums.PaymentStatusLinkGenerated), string(enums.PaymentStatusProcessing), string(enums.PaymentStatusFailed)}

	providerPaymentID, _ := paymentIntent["id"].(string)
	additionalFields := map[string]interface{}{
		"providerPaymentId": providerPaymentID,
	}

	if err := updatePaymentStatus(ctx, paymentID, string(enums.PaymentStatusCanceled), additionalFields, allowedStatuses); err != nil {
		log.Printf(`{"error": "Failed to update payment status", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
			err.Error(), paymentID, traceID)
	}

	payment, _ = getPayment(ctx, paymentID)
	_ = publishPaymentStatusEvent(ctx, payment, string(enums.EventTypePaymentIntentCanceled), additionalFields)

	return map[string]string{"paymentId": paymentID, "status": string(enums.PaymentStatusCanceled)}, nil
}

func handlePaymentIntentProcessing(ctx context.Context, paymentIntent map[string]interface{}, traceID string) (map[string]string, error) {
	metadata, _ := paymentIntent["metadata"].(map[string]interface{})
	paymentID, _ := metadata["paymentId"].(string)

	if paymentID == "" {
		return nil, errors.New("missing paymentId in payment_intent metadata")
	}

	payment, err := getPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	// Idempotency check
	if payment.Status == string(enums.PaymentStatusProcessing) {
		return map[string]string{"paymentId": paymentID, "status": string(enums.PaymentStatusProcessing)}, nil
	}

	// Protected statuses
	protectedStatuses := []string{string(enums.PaymentStatusPaid), string(enums.PaymentStatusRefunded), string(enums.PaymentStatusCanceled), string(enums.PaymentStatusFailed)}
	for _, s := range protectedStatuses {
		if payment.Status == s {
			return map[string]string{"paymentId": paymentID, "status": payment.Status}, nil
		}
	}

	allowedStatuses := []string{string(enums.PaymentStatusLinkGenerated), "created"}

	providerPaymentID, _ := paymentIntent["id"].(string)
	additionalFields := map[string]interface{}{
		"providerPaymentId": providerPaymentID,
	}

	if err := updatePaymentStatus(ctx, paymentID, string(enums.PaymentStatusProcessing), additionalFields, allowedStatuses); err != nil {
		log.Printf(`{"error": "Failed to update payment status", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
			err.Error(), paymentID, traceID)
	}

	payment, _ = getPayment(ctx, paymentID)
	_ = publishPaymentStatusEvent(ctx, payment, string(enums.EventTypePaymentIntentProcessing), additionalFields)

	return map[string]string{"paymentId": paymentID, "status": string(enums.PaymentStatusProcessing)}, nil
}

func handleChargeRefunded(ctx context.Context, charge map[string]interface{}, traceID string) (map[string]string, error) {
	metadata, _ := charge["metadata"].(map[string]interface{})
	paymentID, _ := metadata["paymentId"].(string)

	if paymentID == "" {
		return nil, errors.New("missing paymentId in charge metadata")
	}

	payment, err := getPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	// Idempotency check
	if payment.Status == string(enums.PaymentStatusRefunded) {
		return map[string]string{"paymentId": paymentID, "status": string(enums.PaymentStatusRefunded)}, nil
	}

	// Only allow refund from paid status
	allowedStatuses := []string{string(enums.PaymentStatusPaid)}

	// Extract refund info
	var refundID, refundStatus string
	var refundAmount float64

	refunds, _ := charge["refunds"].(map[string]interface{})
	refundsData, _ := refunds["data"].([]interface{})
	if len(refundsData) > 0 {
		latestRefund, _ := refundsData[0].(map[string]interface{})
		refundID, _ = latestRefund["id"].(string)
		refundStatus, _ = latestRefund["status"].(string)
		amount, _ := latestRefund["amount"].(float64)
		refundAmount = amount / 100
	} else {
		amountRefunded, _ := charge["amount_refunded"].(float64)
		refundAmount = amountRefunded / 100
		isRefunded, _ := charge["refunded"].(bool)
		if isRefunded {
			refundStatus = "succeeded"
		} else {
			refundStatus = "pending"
		}
	}

	additionalFields := map[string]interface{}{
		"refundedAt":   time.Now().UTC().Format(time.RFC3339),
		"refundStatus": refundStatus,
		"refundAmount": refundAmount,
	}
	if refundID != "" {
		additionalFields["refundId"] = refundID
	}

	if err := updatePaymentStatus(ctx, paymentID, string(enums.PaymentStatusRefunded), additionalFields, allowedStatuses); err != nil {
		log.Printf(`{"error": "Failed to update payment status", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
			err.Error(), paymentID, traceID)
	}

	payment, _ = getPayment(ctx, paymentID)
	_ = publishPaymentStatusEvent(ctx, payment, string(enums.EventTypeChargeRefunded), additionalFields)

	return map[string]string{"paymentId": paymentID, "status": string(enums.PaymentStatusRefunded)}, nil
}

// ============================================================================
// MAIN HANDLER
// ============================================================================

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	initClients()

	traceID := request.RequestContext.RequestID
	log.Printf(`{"message": "Webhook received", "traceId": "%s"}`, traceID)

	// Get signature from headers
	signature := request.Headers["stripe-signature"]
	if signature == "" {
		signature = request.Headers["Stripe-Signature"]
	}
	if signature == "" {
		return errorResponse(400, "MissingSignature", "Missing Stripe signature", nil), nil
	}

	body := request.Body
	if body == "" {
		return errorResponse(400, "MissingBody", "Missing request body", nil), nil
	}

	// Extract company name from metadata to get webhook secret
	var bodyJSON map[string]interface{}
	if err := json.Unmarshal([]byte(body), &bodyJSON); err != nil {
		return errorResponse(400, "InvalidJSON", "Invalid JSON in request body", nil), nil
	}

	data, _ := bodyJSON["data"].(map[string]interface{})
	object, _ := data["object"].(map[string]interface{})
	metadata, _ := object["metadata"].(map[string]interface{})
	companyName, _ := metadata["companyName"].(string)

	if companyName == "" {
		return errorResponse(400, "MissingCompanyName", "Missing companyName in metadata", nil), nil
	}

	// Get webhook secret
	keyset, err := getCompanyKeyset(ctx, companyName)
	if err != nil {
		log.Printf(`{"error": "Failed to get webhook secret", "details": "%s", "traceId": "%s"}`, err.Error(), traceID)
		return errorResponse(500, "CredentialRetrievalError", "Failed to retrieve webhook credentials", nil), nil
	}

	// Verify signature
	stripeEvent, err := webhook.ConstructEvent([]byte(body), signature, keyset.WebhookSecret)
	if err != nil {
		log.Printf(`{"error": "Invalid signature", "details": "%s", "traceId": "%s"}`, err.Error(), traceID)
		return errorResponse(400, "InvalidSignature", "Invalid signature", nil), nil
	}

	eventType := stripeEvent.Type
	log.Printf(`{"eventType": "%s", "eventId": "%s", "traceId": "%s"}`, eventType, stripeEvent.ID, traceID)

	// Process event
	var eventData map[string]interface{}
	if err := json.Unmarshal(stripeEvent.Data.Raw, &eventData); err != nil {
		return errorResponse(400, "InvalidEventData", "Failed to parse event data", nil), nil
	}

	var result map[string]string
	var handlerErr error

	switch eventType {
	case "payment_intent.succeeded":
		result, handlerErr = handlePaymentIntentSucceeded(ctx, eventData, traceID)
	case "payment_intent.payment_failed":
		result, handlerErr = handlePaymentIntentFailed(ctx, eventData, traceID)
	case "payment_intent.canceled":
		result, handlerErr = handlePaymentIntentCanceled(ctx, eventData, traceID)
	case "payment_intent.processing":
		result, handlerErr = handlePaymentIntentProcessing(ctx, eventData, traceID)
	case "charge.refunded":
		result, handlerErr = handleChargeRefunded(ctx, eventData, traceID)
	default:
		log.Printf(`{"message": "Unhandled event type", "eventType": "%s", "traceId": "%s"}`, eventType, traceID)
		handled := false
		return successResponse(200, SuccessResponse{
			Received:  true,
			EventType: string(eventType),
			Handled:   &handled,
		}), nil
	}

	if handlerErr != nil {
		log.Printf(`{"error": "Handler error", "details": "%s", "traceId": "%s"}`, handlerErr.Error(), traceID)
		return errorResponse(400, "WebhookError", "Webhook processing error", map[string]interface{}{"message": handlerErr.Error()}), nil
	}

	log.Printf(`{"message": "Event processed", "eventType": "%s", "result": %v}`, eventType, result)

	return successResponse(200, SuccessResponse{
		Received:  true,
		EventType: string(eventType),
		Result:    result,
	}), nil
}

func main() {
	lambda.Start(handler)
}
