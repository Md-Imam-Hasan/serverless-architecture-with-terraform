package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
)

type PaymentRequest struct {
	OrderNumber         string  `json:"orderNumber"`
	CompanyID           string  `json:"companyId"`
	PaymentMode         string  `json:"paymentMode"`
	PaymentType         string  `json:"paymentType"`
	Provider            string  `json:"provider"`
	Amount              float64 `json:"amount"`
	Currency            string  `json:"currency"`
	SourceKey           string  `json:"sourceKey"`
	ProviderPaymentID   *string `json:"providerPaymentId,omitempty"`
	PaymentLinkID       *string `json:"paymentLinkId,omitempty"`
	PaymentLink         *string `json:"paymentLink,omitempty"`
	CancelURL           *string `json:"cancelUrl,omitempty"`
	SuccessURL          *string `json:"successUrl,omitempty"`
	ReceiveURL          *string `json:"receiveUrl,omitempty"`
	TripID              *string `json:"tripId,omitempty"`
	ScheduledAt         *string `json:"scheduledAt,omitempty"`
	ExpiresAt           *string `json:"expiresAt,omitempty"`
	PaidAt              *string `json:"paidAt,omitempty"`
	RetryAfterTimestamp *string `json:"retryAfterTimestamp,omitempty"`
	CallbackURL         *string `json:"callbackUrl,omitempty"`
}

type PaymentItem struct {
	PK                  string  `dynamodbav:"PK"`
	SK                  string  `dynamodbav:"SK"`
	PaymentID           string  `dynamodbav:"paymentId"`
	OrderNumber         string  `dynamodbav:"orderNumber"`
	CompanyID           string  `dynamodbav:"companyId"`
	Status              string  `dynamodbav:"status"`
	PaymentMode         string  `dynamodbav:"paymentMode"`
	PaymentType         string  `dynamodbav:"paymentType"`
	Provider            string  `dynamodbav:"provider"`
	Amount              float64 `dynamodbav:"amount"`
	Currency            string  `dynamodbav:"currency"`
	SourceKey           string  `dynamodbav:"sourceKey"`
	CreatedAt           string  `dynamodbav:"createdAt"`
	UpdatedAt           string  `dynamodbav:"updatedAt"`
	ProviderPaymentID   *string `dynamodbav:"providerPaymentId,omitempty"`
	PaymentLinkID       *string `dynamodbav:"paymentLinkId,omitempty"`
	PaymentLink         *string `dynamodbav:"paymentLink,omitempty"`
	CancelURL           *string `dynamodbav:"cancelUrl,omitempty"`
	SuccessURL          *string `dynamodbav:"successUrl,omitempty"`
	ReceiveURL          *string `dynamodbav:"receiveUrl,omitempty"`
	TripID              *string `dynamodbav:"tripId,omitempty"`
	ScheduledAt         *string `dynamodbav:"scheduledAt,omitempty"`
	ExpiresAt           *string `dynamodbav:"expiresAt,omitempty"`
	PaidAt              *string `dynamodbav:"paidAt,omitempty"`
	RetryAfterTimestamp *string `dynamodbav:"retryAfterTimestamp,omitempty"`
	CallbackURL         *string `dynamodbav:"callbackUrl,omitempty"`
}

type PaymentResponse struct {
	PaymentID string `json:"paymentId"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	TraceID   string `json:"traceId"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	TraceID string `json:"traceId"`
}

var (
	dynamoClient *dynamodb.Client
	tableName    string
)

func init() {
	tableName = os.Getenv("PAYMENTS_TABLE_NAME")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	dynamoClient = dynamodb.NewFromConfig(cfg)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	traceID := request.RequestContext.RequestID

	// Log the incoming event
	eventJSON, _ := json.Marshal(map[string]interface{}{
		"event":   request,
		"traceId": traceID,
	})
	log.Printf("%s", eventJSON)

	// Parse request body
	var paymentReq PaymentRequest
	if err := json.Unmarshal([]byte(request.Body), &paymentReq); err != nil {
		log.Printf(`{"error": "%s", "traceId": "%s"}`, err.Error(), traceID)
		return errorResponse(500, "Internal server error", traceID), nil
	}

	// Generate payment ID and timestamp
	paymentID := uuid.New().String()
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Build payment item
	paymentItem := PaymentItem{
		PK:                  fmt.Sprintf("PAYMENT#%s", paymentID),
		SK:                  "METADATA",
		PaymentID:           paymentID,
		OrderNumber:         paymentReq.OrderNumber,
		CompanyID:           paymentReq.CompanyID,
		Status:              "created",
		PaymentMode:         paymentReq.PaymentMode,
		PaymentType:         paymentReq.PaymentType,
		Provider:            paymentReq.Provider,
		Amount:              paymentReq.Amount,
		Currency:            paymentReq.Currency,
		SourceKey:           paymentReq.SourceKey,
		CreatedAt:           timestamp,
		UpdatedAt:           timestamp,
		ProviderPaymentID:   paymentReq.ProviderPaymentID,
		PaymentLinkID:       paymentReq.PaymentLinkID,
		PaymentLink:         paymentReq.PaymentLink,
		CancelURL:           paymentReq.CancelURL,
		SuccessURL:          paymentReq.SuccessURL,
		ReceiveURL:          paymentReq.ReceiveURL,
		TripID:              paymentReq.TripID,
		ScheduledAt:         paymentReq.ScheduledAt,
		ExpiresAt:           paymentReq.ExpiresAt,
		PaidAt:              paymentReq.PaidAt,
		RetryAfterTimestamp: paymentReq.RetryAfterTimestamp,
		CallbackURL:         paymentReq.CallbackURL,
	}

	// Convert to DynamoDB attribute values
	av, err := attributevalue.MarshalMap(paymentItem)
	if err != nil {
		log.Printf(`{"error": "%s", "traceId": "%s"}`, err.Error(), traceID)
		return errorResponse(500, "Internal server error", traceID), nil
	}

	// Write to DynamoDB
	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      av,
	})
	if err != nil {
		log.Printf(`{"error": "%s", "traceId": "%s"}`, err.Error(), traceID)
		return errorResponse(500, "Internal server error", traceID), nil
	}

	// Log success
	successLog, _ := json.Marshal(map[string]interface{}{
		"message":   "Payment created",
		"paymentId": paymentID,
		"traceId":   traceID,
	})
	log.Printf("%s", successLog)

	// Build response
	response := PaymentResponse{
		PaymentID: paymentID,
		Status:    "created",
		CreatedAt: timestamp,
		TraceID:   traceID,
	}

	responseBody, _ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		StatusCode: 201,
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Headers": "Content-Type,Authorization",
			"Access-Control-Allow-Methods": "OPTIONS,POST,GET",
		},
		Body: string(responseBody),
	}, nil
}

func errorResponse(statusCode int, message, traceID string) events.APIGatewayProxyResponse {
	errorResp := ErrorResponse{
		Error:   message,
		TraceID: traceID,
	}

	body, _ := json.Marshal(errorResp)

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(body),
	}
}

func main() {
	lambda.Start(handler)
}
