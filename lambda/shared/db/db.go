package db

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	dynamoClient     *dynamodb.Client
	dynamoClientOnce sync.Once

	ErrPaymentNotFound         = errors.New("payment not found")
	ErrDatabaseError           = errors.New("database error")
	ErrTransactionCancelled    = errors.New("transaction cancelled")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)

func getDynamoClient(ctx context.Context) (*dynamodb.Client, error) {
	var initErr error
	dynamoClientOnce.Do(func() {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			initErr = fmt.Errorf("unable to load SDK config: %w", err)
			return
		}
		dynamoClient = dynamodb.NewFromConfig(cfg)
	})
	return dynamoClient, initErr
}

// PaymentRecord represents a payment in DynamoDB
type PaymentRecord struct {
	PK                string   `dynamodbav:"PK"`
	SK                string   `dynamodbav:"SK"`
	PaymentID         string   `dynamodbav:"paymentId"`
	OrderNumber       string   `dynamodbav:"orderNumber"`
	CompanyName       string   `dynamodbav:"companyName"`
	Status            string   `dynamodbav:"status"`
	PaymentMode       string   `dynamodbav:"paymentMode"`
	PaymentType       string   `dynamodbav:"paymentType"`
	Provider          string   `dynamodbav:"provider"`
	Amount            int      `dynamodbav:"amount"` // Stored in cents
	Currency          string   `dynamodbav:"currency"`
	SourceKey         string   `dynamodbav:"sourceKey"`
	CreatedAt         string   `dynamodbav:"createdAt"`
	UpdatedAt         string   `dynamodbav:"updatedAt"`
	ProviderPaymentID *string  `dynamodbav:"providerPaymentId,omitempty"`
	PaymentLinkID     *string  `dynamodbav:"paymentLinkId,omitempty"`
	PaymentLink       *string  `dynamodbav:"paymentLink,omitempty"`
	TripID            *string  `dynamodbav:"tripId,omitempty"`
	ScheduledAt       *string  `dynamodbav:"scheduledAt,omitempty"`
	RequestID         *string  `dynamodbav:"requestId,omitempty"`
	PaidAt            *string  `dynamodbav:"paidAt,omitempty"`
	FailureReason     *string  `dynamodbav:"failureReason,omitempty"`
	FailureCount      *int     `dynamodbav:"failureCount,omitempty"`
	LastFailedAt      *string  `dynamodbav:"lastFailedAt,omitempty"`
	RefundID          *string  `dynamodbav:"refundId,omitempty"`
	RefundedAt        *string  `dynamodbav:"refundedAt,omitempty"`
	RefundStatus      *string  `dynamodbav:"refundStatus,omitempty"`
	RefundAmount      *float64 `dynamodbav:"refundAmount,omitempty"`
	AmountReceived    *float64 `dynamodbav:"amountReceived,omitempty"`
	FailureMessage    *string  `dynamodbav:"failureMessage,omitempty"`
}

// GetPayment retrieves a payment by ID
func GetPayment(ctx context.Context, tableName, paymentID string) (*PaymentRecord, error) {
	client, err := getDynamoClient(ctx)
	if err != nil {
		return nil, err
	}

	result, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PAYMENT#%s", paymentID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}

	if result.Item == nil {
		return nil, ErrPaymentNotFound
	}

	var payment PaymentRecord
	if err := attributevalue.UnmarshalMap(result.Item, &payment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment: %w", err)
	}

	return &payment, nil
}

// SavePayment saves a single payment record
func SavePayment(ctx context.Context, tableName string, payment *PaymentRecord) error {
	client, err := getDynamoClient(ctx)
	if err != nil {
		return err
	}

	av, err := attributevalue.MarshalMap(payment)
	if err != nil {
		return fmt.Errorf("failed to marshal payment: %w", err)
	}

	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}

	return nil
}

// SavePaymentsTransactional saves multiple payments in a transaction (max 100)
func SavePaymentsTransactional(ctx context.Context, tableName string, payments []*PaymentRecord) error {
	if len(payments) == 0 {
		return nil
	}
	if len(payments) > 100 {
		return fmt.Errorf("transaction limit exceeded: max 100 items, got %d", len(payments))
	}

	client, err := getDynamoClient(ctx)
	if err != nil {
		return err
	}

	transactItems := make([]types.TransactWriteItem, len(payments))
	for i, payment := range payments {
		av, err := attributevalue.MarshalMap(payment)
		if err != nil {
			return fmt.Errorf("failed to marshal payment %s: %w", payment.PaymentID, err)
		}

		transactItems[i] = types.TransactWriteItem{
			Put: &types.Put{
				TableName:           aws.String(tableName),
				Item:                av,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			},
		}
	}

	_, err = client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})
	if err != nil {
		var txErr *types.TransactionCanceledException
		if errors.As(err, &txErr) {
			return fmt.Errorf("%w: %v", ErrTransactionCancelled, err)
		}
		return fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}

	return nil
}

// SavePaymentsInBatches saves payments in batches of specified size
func SavePaymentsInBatches(ctx context.Context, tableName string, payments []*PaymentRecord, batchSize int) error {
	for i := 0; i < len(payments); i += batchSize {
		end := i + batchSize
		if end > len(payments) {
			end = len(payments)
		}
		batch := payments[i:end]
		if err := SavePaymentsTransactional(ctx, tableName, batch); err != nil {
			return err
		}
	}
	return nil
}

// UpdatePaymentStatus updates payment status with additional fields
func UpdatePaymentStatus(ctx context.Context, tableName, paymentID, status string, additionalFields map[string]interface{}) error {
	client, err := getDynamoClient(ctx)
	if err != nil {
		return err
	}

	updateExpr := "SET #status = :status, #updatedAt = :updatedAt"
	exprAttrNames := map[string]string{
		"#status":    "status",
		"#updatedAt": "updatedAt",
	}
	exprAttrValues := map[string]types.AttributeValue{
		":status":    &types.AttributeValueMemberS{Value: status},
		":updatedAt": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
	}

	// Add additional fields to update expression
	i := 0
	for key, value := range additionalFields {
		if key == "status" || key == "updatedAt" {
			continue
		}
		placeholder := fmt.Sprintf("#field%d", i)
		valuePlaceholder := fmt.Sprintf(":val%d", i)
		updateExpr += fmt.Sprintf(", %s = %s", placeholder, valuePlaceholder)
		exprAttrNames[placeholder] = key

		// Convert value to AttributeValue
		av, err := attributevalue.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal field %s: %w", key, err)
		}
		exprAttrValues[valuePlaceholder] = av
		i++
	}

	_, err = client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PAYMENT#%s", paymentID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}

	return nil
}

// UpdatePaymentStatusConditional updates payment status only if current status is in allowed list
func UpdatePaymentStatusConditional(ctx context.Context, tableName, paymentID, status string, additionalFields map[string]interface{}, allowedCurrentStatuses []string) error {
	client, err := getDynamoClient(ctx)
	if err != nil {
		return err
	}

	updateExpr := "SET #status = :status, #updatedAt = :updatedAt"
	exprAttrNames := map[string]string{
		"#status":    "status",
		"#updatedAt": "updatedAt",
	}
	exprAttrValues := map[string]types.AttributeValue{
		":status":    &types.AttributeValueMemberS{Value: status},
		":updatedAt": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
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

	// Build condition expression for allowed statuses
	conditionExpr := "#status IN ("
	for j, allowedStatus := range allowedCurrentStatuses {
		placeholder := fmt.Sprintf(":allowedStatus%d", j)
		if j > 0 {
			conditionExpr += ", "
		}
		conditionExpr += placeholder
		exprAttrValues[placeholder] = &types.AttributeValueMemberS{Value: allowedStatus}
	}
	conditionExpr += ")"

	_, err = client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PAYMENT#%s", paymentID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression:          aws.String(updateExpr),
		ConditionExpression:       aws.String(conditionExpr),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
	})
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return ErrInvalidStatusTransition
		}
		return fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}

	return nil
}
