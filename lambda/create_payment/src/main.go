package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentlink"
	"github.com/stripe/stripe-go/v76/price"
	"github.com/stripe/stripe-go/v76/product"
)

// ============================================================================
// TYPES AND CONSTANTS
// ============================================================================

// PaymentMode enum
type PaymentMode string

const (
	PaymentModeOneTime      PaymentMode = "one_time"
	PaymentModeSubscription PaymentMode = "subscription"
)

func (m PaymentMode) IsValid() bool {
	return m == PaymentModeOneTime || m == PaymentModeSubscription
}

// PaymentType enum
type PaymentType string

const (
	PaymentTypeCard         PaymentType = "card"
	PaymentTypeBankTransfer PaymentType = "bank_transfer"
	PaymentTypeWallet       PaymentType = "wallet"
)

func (t PaymentType) IsValid() bool {
	return t == PaymentTypeCard || t == PaymentTypeBankTransfer || t == PaymentTypeWallet
}

// PaymentStatus enum
type PaymentStatus string

const (
	PaymentStatusPendingLink     PaymentStatus = "pending_link"
	PaymentStatusLinkGenerated   PaymentStatus = "link_generated"
	PaymentStatusScheduled       PaymentStatus = "scheduled"
	PaymentStatusScheduledFailed PaymentStatus = "scheduled_failed"
)

// Request/Response types
type CreatePaymentRequest struct {
	CompanyName string  `json:"companyName"`
	SourceKey   string  `json:"sourceKey"`
	PaymentMode string  `json:"paymentMode"`
	Orders      []Order `json:"orders"`
}

type Order struct {
	OrderNumber string  `json:"orderNumber"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	PaymentType string  `json:"paymentType"`
	ScheduledAt *string `json:"scheduledAt,omitempty"`
	TripID      *string `json:"tripId,omitempty"`
	SuccessURL  *string `json:"successUrl,omitempty"`
	RequestID   *string `json:"requestId,omitempty"`
}

type PaymentRecord struct {
	PK                string  `dynamodbav:"PK"`
	SK                string  `dynamodbav:"SK"`
	PaymentID         string  `dynamodbav:"paymentId"`
	OrderNumber       string  `dynamodbav:"orderNumber"`
	CompanyID         string  `dynamodbav:"companyId"` // GSI hash key
	CompanyName       string  `dynamodbav:"companyName"`
	Status            string  `dynamodbav:"status"`
	PaymentMode       string  `dynamodbav:"paymentMode"`
	PaymentType       string  `dynamodbav:"paymentType"`
	Provider          string  `dynamodbav:"provider"`
	Amount            int     `dynamodbav:"amount"` // Stored in cents
	Currency          string  `dynamodbav:"currency"`
	SourceKey         string  `dynamodbav:"sourceKey"`
	CreatedAt         string  `dynamodbav:"createdAt"`
	UpdatedAt         string  `dynamodbav:"updatedAt"`
	TripID            *string `dynamodbav:"tripId,omitempty"`
	ScheduledAt       *string `dynamodbav:"scheduledAt,omitempty"`
	RequestID         *string `dynamodbav:"requestId,omitempty"`
	PaymentLink       *string `dynamodbav:"paymentLink,omitempty"`
	PaymentLinkID     *string `dynamodbav:"paymentLinkId,omitempty"`
	ProviderPaymentID *string `dynamodbav:"providerPaymentId,omitempty"`
}

type PaymentResult struct {
	PaymentID         string  `json:"paymentId"`
	OrderNumber       string  `json:"orderNumber"`
	RequestID         *string `json:"requestId,omitempty"`
	Amount            float64 `json:"amount"`
	PaymentLink       *string `json:"paymentLink,omitempty"`
	PaymentLinkID     *string `json:"paymentLinkId,omitempty"`
	ProviderPaymentID *string `json:"providerPaymentId,omitempty"`
	Message           string  `json:"message"`
	Status            string  `json:"status"`
}

type SuccessResponse struct {
	Payments    []PaymentResult `json:"payments"`
	CreatedAt   string          `json:"createdAt"`
	PaymentMode string          `json:"paymentMode"`
}

type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type CompanyKeyset struct {
	SecretKey      string `json:"secret_key"`
	PublishableKey string `json:"publishable_key"`
	WebhookSecret  string `json:"webhook_secret"`
}

// Internal types for processing
type immediatePayment struct {
	record      *PaymentRecord
	redirectURL *string
}

type scheduledPayment struct {
	record      *PaymentRecord
	redirectURL *string
}

// ============================================================================
// GLOBAL VARIABLES
// ============================================================================

var (
	dynamoClient    *dynamodb.Client
	sfnClient       *sfn.Client
	ssmClient       *ssm.Client
	tableName       string
	stateMachineARN string
	environment     string
	initOnce        sync.Once
)

func initClients() {
	initOnce.Do(func() {
		tableName = os.Getenv("PAYMENTS_TABLE_NAME")
		stateMachineARN = os.Getenv("STATE_MACHINE_ARN")
		environment = os.Getenv("ENVIRONMENT")

		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatalf("unable to load SDK config: %v", err)
		}

		dynamoClient = dynamodb.NewFromConfig(cfg)
		sfnClient = sfn.NewFromConfig(cfg)
		ssmClient = ssm.NewFromConfig(cfg)
	})
}

// ============================================================================
// VALIDATION FUNCTIONS
// ============================================================================

func validatePaymentMode(paymentMode string) (PaymentMode, *ErrorResponse) {
	pm := PaymentMode(paymentMode)
	if !pm.IsValid() {
		return "", &ErrorResponse{
			Error:   "InvalidPaymentMode",
			Message: fmt.Sprintf("paymentMode must be one of: %s, %s", PaymentModeOneTime, PaymentModeSubscription),
		}
	}
	return pm, nil
}

func validatePaymentType(paymentType string) *ErrorResponse {
	pt := PaymentType(paymentType)
	if !pt.IsValid() {
		return &ErrorResponse{
			Error:   "InvalidPaymentType",
			Message: fmt.Sprintf("paymentType must be one of: %s, %s, %s", PaymentTypeCard, PaymentTypeBankTransfer, PaymentTypeWallet),
		}
	}
	return nil
}

func validateAmount(amount float64, orderNumber string) *ErrorResponse {
	if amount <= 0 {
		return &ErrorResponse{
			Error:   "InvalidAmount",
			Message: fmt.Sprintf("Amount must be greater than 0 for order %s", orderNumber),
		}
	}
	return nil
}

// ============================================================================
// PHASE 1: BUILD PAYMENT RECORDS
// ============================================================================

func buildPaymentRecords(orders []Order, companyName, sourceKey string, paymentMode PaymentMode) ([]*PaymentRecord, []immediatePayment, []scheduledPayment, *ErrorResponse) {
	var paymentRecords []*PaymentRecord
	var immediatePayments []immediatePayment
	var scheduledPayments []scheduledPayment

	now := time.Now().UTC().Format(time.RFC3339)

	for _, order := range orders {
		// Validate required order fields
		if order.OrderNumber == "" || order.Currency == "" || order.PaymentType == "" {
			return nil, nil, nil, &ErrorResponse{
				Error:   "MissingRequiredField",
				Message: "Missing required field in order: orderNumber, currency, or paymentType",
			}
		}

		// Validate paymentType
		if err := validatePaymentType(order.PaymentType); err != nil {
			return nil, nil, nil, err
		}

		// Validate amount
		if err := validateAmount(order.Amount, order.OrderNumber); err != nil {
			return nil, nil, nil, err
		}

		paymentID := uuid.New().String()
		isScheduled := false

		// Check if payment is scheduled for future
		if order.ScheduledAt != nil && *order.ScheduledAt != "" {
			scheduledTime, err := time.Parse(time.RFC3339, strings.Replace(*order.ScheduledAt, "Z", "+00:00", 1))
			if err != nil {
				scheduledTime, err = time.Parse(time.RFC3339, *order.ScheduledAt)
			}
			if err == nil {
				currentTime := time.Now().UTC()
				if scheduledTime.After(currentTime) {
					isScheduled = true
				} else if scheduledTime.Before(currentTime) {
					return nil, nil, nil, &ErrorResponse{
						Error:   "InvalidScheduledAt",
						Message: fmt.Sprintf("scheduledAt is in the past for order %s", order.OrderNumber),
					}
				}
			}
		}

		// Determine initial status
		status := string(PaymentStatusPendingLink)
		if isScheduled {
			status = string(PaymentStatusScheduled)
		}

		// Build payment record
		record := &PaymentRecord{
			PK:          fmt.Sprintf("PAYMENT#%s", paymentID),
			SK:          "METADATA",
			PaymentID:   paymentID,
			OrderNumber: order.OrderNumber,
			CompanyID:   companyName, // GSI hash key
			CompanyName: companyName,
			Status:      status,
			PaymentMode: string(paymentMode),
			PaymentType: order.PaymentType,
			Provider:    "stripe",
			Amount:      int(order.Amount * 100), // Store in cents
			Currency:    order.Currency,
			SourceKey:   sourceKey,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Add optional fields
		if order.TripID != nil {
			record.TripID = order.TripID
		}
		if order.ScheduledAt != nil {
			record.ScheduledAt = order.ScheduledAt
		}
		if order.RequestID != nil {
			record.RequestID = order.RequestID
		}

		paymentRecords = append(paymentRecords, record)

		if isScheduled {
			scheduledPayments = append(scheduledPayments, scheduledPayment{
				record:      record,
				redirectURL: order.SuccessURL,
			})
		} else {
			immediatePayments = append(immediatePayments, immediatePayment{
				record:      record,
				redirectURL: order.SuccessURL,
			})
		}
	}

	return paymentRecords, immediatePayments, scheduledPayments, nil
}

// ============================================================================
// PHASE 2: DATABASE OPERATIONS
// ============================================================================

func persistPaymentRecords(ctx context.Context, records []*PaymentRecord) *ErrorResponse {
	if len(records) == 0 {
		return nil
	}

	log.Printf(`{"message": "Saving %d payments transactionally"}`, len(records))

	if len(records) <= 100 {
		return savePaymentsTransactional(ctx, records)
	}

	// Batch into chunks of 100
	for i := 0; i < len(records); i += 100 {
		end := i + 100
		if end > len(records) {
			end = len(records)
		}
		if err := savePaymentsTransactional(ctx, records[i:end]); err != nil {
			return err
		}
	}

	log.Printf(`{"message": "Successfully saved %d payments to DynamoDB"}`, len(records))
	return nil
}

func savePaymentsTransactional(ctx context.Context, records []*PaymentRecord) *ErrorResponse {
	transactItems := make([]types.TransactWriteItem, len(records))

	for i, record := range records {
		av, err := attributevalue.MarshalMap(record)
		if err != nil {
			return &ErrorResponse{
				Error:   "DatabaseError",
				Message: fmt.Sprintf("Failed to marshal payment %s", record.PaymentID),
			}
		}

		transactItems[i] = types.TransactWriteItem{
			Put: &types.Put{
				TableName:           aws.String(tableName),
				Item:                av,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			},
		}
	}

	_, err := dynamoClient.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})
	if err != nil {
		var txErr *types.TransactionCanceledException
		if errors.As(err, &txErr) {
			return &ErrorResponse{
				Error:   "DuplicatePayment",
				Message: fmt.Sprintf("One or more payments already exist: %v", err),
			}
		}
		return &ErrorResponse{
			Error:   "DatabaseError",
			Message: "Failed to save payments to database. Please try again later.",
		}
	}

	return nil
}

func updatePaymentStatus(ctx context.Context, paymentID, status string, additionalFields map[string]interface{}) error {
	updateExpr := "SET #status = :status, #updatedAt = :updatedAt"
	exprAttrNames := map[string]string{
		"#status":    "status",
		"#updatedAt": "updatedAt",
	}
	exprAttrValues := map[string]types.AttributeValue{
		":status":    &types.AttributeValueMemberS{Value: status},
		":updatedAt": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
	}

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

	_, err := dynamoClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PAYMENT#%s", paymentID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
	})

	return err
}

// ============================================================================
// PHASE 3: PARAMETER STORE & STRIPE
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

func getCompanySuccessURL(companyName string) string {
	return fmt.Sprintf("https://%s/api/payment/success", companyName)
}

func generateStripeLinks(ctx context.Context, payments []immediatePayment, companyName, sourceKey, traceID string) []PaymentResult {
	var results []PaymentResult

	for _, payment := range payments {
		record := payment.record
		redirectURL := payment.redirectURL

		result := PaymentResult{
			PaymentID:   record.PaymentID,
			OrderNumber: record.OrderNumber,
			RequestID:   record.RequestID,
			Amount:      float64(record.Amount) / 100,
		}

		// Get Stripe keys
		keyset, err := getCompanyKeyset(ctx, companyName)
		if err != nil {
			log.Printf(`{"error": "Failed to get company keyset", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
				err.Error(), record.PaymentID, traceID)
			result.Message = "Payment saved but link generation failed. Please retry."
			result.Status = string(PaymentStatusPendingLink)
			results = append(results, result)
			continue
		}

		// Get redirect URL from company name if not provided
		if redirectURL == nil || *redirectURL == "" {
			url := getCompanySuccessURL(companyName)
			redirectURL = &url
		}

		// Generate Stripe payment link
		stripeResult, err := createStripePaymentLink(
			record.PaymentID,
			companyName,
			record.Amount,
			record.Currency,
			fmt.Sprintf("Payment for order %s", record.OrderNumber),
			keyset.SecretKey,
			sourceKey,
			redirectURL,
			map[string]string{"orderNumber": record.OrderNumber},
		)
		if err != nil {
			log.Printf(`{"error": "Error generating payment link", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
				err.Error(), record.PaymentID, traceID)
			result.Message = "Payment saved but link generation failed. Please retry."
			result.Status = string(PaymentStatusPendingLink)
			results = append(results, result)
			continue
		}

		// Update payment record with Stripe link info
		err = updatePaymentStatus(ctx, record.PaymentID, string(PaymentStatusLinkGenerated), map[string]interface{}{
			"paymentLink":       stripeResult.paymentLink,
			"paymentLinkId":     stripeResult.paymentLinkID,
			"providerPaymentId": stripeResult.providerPaymentID,
		})
		if err != nil {
			log.Printf(`{"error": "Failed to update payment status", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
				err.Error(), record.PaymentID, traceID)
		}

		result.PaymentLink = &stripeResult.paymentLink
		result.PaymentLinkID = &stripeResult.paymentLinkID
		result.ProviderPaymentID = &stripeResult.providerPaymentID
		result.Message = "Payment link generated successfully"
		result.Status = string(PaymentStatusLinkGenerated)
		results = append(results, result)
	}

	return results
}

type stripeResult struct {
	paymentLink       string
	paymentLinkID     string
	providerPaymentID string
}

func createStripePaymentLink(paymentID, companyName string, amount int, currency, description, secretKey, sourceKey string, redirectURL *string, metadata map[string]string) (*stripeResult, error) {
	stripe.Key = secretKey

	// Create product
	productParams := &stripe.ProductParams{
		Name: stripe.String(description),
	}
	productParams.AddMetadata("paymentId", paymentID)
	productParams.AddMetadata("companyName", companyName)
	productParams.AddMetadata("sourceKey", sourceKey)
	for k, v := range metadata {
		productParams.AddMetadata(k, v)
	}

	prod, err := product.New(productParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Create price
	priceParams := &stripe.PriceParams{
		Product:    stripe.String(prod.ID),
		UnitAmount: stripe.Int64(int64(amount)),
		Currency:   stripe.String(currency),
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

	linkParams.AddMetadata("paymentId", paymentID)
	linkParams.AddMetadata("companyName", companyName)
	linkParams.AddMetadata("sourceKey", sourceKey)
	for k, v := range metadata {
		linkParams.AddMetadata(k, v)
	}

	linkParams.PaymentIntentData = &stripe.PaymentLinkPaymentIntentDataParams{}
	linkParams.PaymentIntentData.AddMetadata("paymentId", paymentID)
	linkParams.PaymentIntentData.AddMetadata("companyName", companyName)
	linkParams.PaymentIntentData.AddMetadata("sourceKey", sourceKey)
	for k, v := range metadata {
		linkParams.PaymentIntentData.AddMetadata(k, v)
	}

	link, err := paymentlink.New(linkParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment link: %w", err)
	}

	return &stripeResult{
		paymentLink:       link.URL,
		paymentLinkID:     link.ID,
		providerPaymentID: prod.ID,
	}, nil
}

// ============================================================================
// PHASE 4: STEP FUNCTIONS SCHEDULING
// ============================================================================

func schedulePayments(ctx context.Context, payments []scheduledPayment, traceID string) []PaymentResult {
	var results []PaymentResult

	for _, payment := range payments {
		record := payment.record

		result := PaymentResult{
			PaymentID:   record.PaymentID,
			OrderNumber: record.OrderNumber,
			RequestID:   record.RequestID,
			Amount:      float64(record.Amount) / 100,
		}

		// Build execution input
		executionInput := map[string]interface{}{
			"PK":          record.PK,
			"SK":          record.SK,
			"paymentId":   record.PaymentID,
			"orderNumber": record.OrderNumber,
			"companyName": record.CompanyName,
			"status":      record.Status,
			"paymentMode": record.PaymentMode,
			"paymentType": record.PaymentType,
			"provider":    record.Provider,
			"amount":      record.Amount,
			"currency":    record.Currency,
			"sourceKey":   record.SourceKey,
			"createdAt":   record.CreatedAt,
			"updatedAt":   record.UpdatedAt,
		}
		if record.TripID != nil {
			executionInput["tripId"] = *record.TripID
		}
		if record.ScheduledAt != nil {
			executionInput["scheduledAt"] = *record.ScheduledAt
		}
		if record.RequestID != nil {
			executionInput["requestId"] = *record.RequestID
		}

		inputJSON, _ := json.Marshal(executionInput)

		executionName := fmt.Sprintf("%s_%d", record.PaymentID, time.Now().UTC().Unix())

		_, err := sfnClient.StartExecution(ctx, &sfn.StartExecutionInput{
			StateMachineArn: aws.String(stateMachineARN),
			Name:            aws.String(executionName),
			Input:           aws.String(string(inputJSON)),
		})
		if err != nil {
			log.Printf(`{"error": "Failed to start Step Function", "details": "%s", "paymentId": "%s", "traceId": "%s"}`,
				err.Error(), record.PaymentID, traceID)
			result.Message = "Payment saved but scheduling failed. Please contact support."
			result.Status = string(PaymentStatusScheduledFailed)
			results = append(results, result)
			continue
		}

		log.Printf(`{"message": "Started Step Function execution", "paymentId": "%s", "traceId": "%s"}`,
			record.PaymentID, traceID)

		result.Message = "Payment scheduled successfully. Payment link will be generated at scheduled time."
		result.Status = string(PaymentStatusScheduled)
		results = append(results, result)
	}

	return results
}

// ============================================================================
// RESPONSE HELPERS
// ============================================================================

var defaultHeaders = map[string]string{
	"Content-Type":                     "application/json",
	"Access-Control-Allow-Origin":      "*",
	"Access-Control-Allow-Headers":     "Content-Type,Authorization,X-Api-Key",
	"Access-Control-Allow-Methods":     "OPTIONS,POST,GET,PUT,PATCH,DELETE",
	"Access-Control-Allow-Credentials": "false",
}

func errorResponse(statusCode int, errResp *ErrorResponse) events.APIGatewayProxyResponse {
	body, _ := json.Marshal(errResp)
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
// MAIN HANDLER
// ============================================================================

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	initClients()

	traceID := request.RequestContext.RequestID
	log.Printf(`{"event": %s, "traceId": "%s"}`, request.Body, traceID)

	// Parse request body
	var req CreatePaymentRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		log.Printf(`{"error": "%s", "traceId": "%s"}`, err.Error(), traceID)
		return errorResponse(400, &ErrorResponse{
			Error:   "InvalidJSON",
			Message: "Invalid JSON in request body",
		}), nil
	}

	// Validate required fields
	if req.CompanyName == "" || req.SourceKey == "" || req.PaymentMode == "" || len(req.Orders) == 0 {
		return errorResponse(400, &ErrorResponse{
			Error:   "MissingRequiredField",
			Message: "Missing required field: companyName, sourceKey, paymentMode, or orders",
		}), nil
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Validate payment mode
	paymentMode, modeErr := validatePaymentMode(req.PaymentMode)
	if modeErr != nil {
		return errorResponse(400, modeErr), nil
	}

	// Phase 1: Build payment records
	paymentRecords, immediatePayments, scheduledPayments, buildErr := buildPaymentRecords(
		req.Orders, req.CompanyName, req.SourceKey, paymentMode,
	)
	if buildErr != nil {
		return errorResponse(400, buildErr), nil
	}

	log.Printf(`{"message": "Built %d payment records (%d immediate, %d scheduled)"}`,
		len(paymentRecords), len(immediatePayments), len(scheduledPayments))

	// Phase 2: Save all payment records transactionally
	if persistErr := persistPaymentRecords(ctx, paymentRecords); persistErr != nil {
		statusCode := 500
		if persistErr.Error == "DuplicatePayment" {
			statusCode = 400
		}
		return errorResponse(statusCode, persistErr), nil
	}

	// Phase 3: Generate Stripe payment links for immediate payments
	immediateResults := generateStripeLinks(ctx, immediatePayments, req.CompanyName, req.SourceKey, traceID)

	// Phase 4: Start Step Functions for scheduled payments
	scheduledResults := schedulePayments(ctx, scheduledPayments, traceID)

	// Combine results
	allResults := append(immediateResults, scheduledResults...)

	return successResponse(201, SuccessResponse{
		Payments:    allResults,
		CreatedAt:   timestamp,
		PaymentMode: req.PaymentMode,
	}), nil
}

func main() {
	lambda.Start(handler)
}
