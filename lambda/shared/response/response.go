package response

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

// ErrorBody represents an error response body
type ErrorBody struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// SuccessBody represents a success response body
type SuccessBody struct {
	Data interface{} `json:"data,omitempty"`
}

var defaultHeaders = map[string]string{
	"Content-Type":                     "application/json",
	"Access-Control-Allow-Origin":      "*",
	"Access-Control-Allow-Headers":     "Content-Type,Authorization,X-Api-Key",
	"Access-Control-Allow-Methods":     "OPTIONS,POST,GET,PUT,PATCH,DELETE",
	"Access-Control-Allow-Credentials": "false",
}

// ErrorResponse creates an API Gateway error response
func ErrorResponse(errorCode, message string, statusCode int, details map[string]interface{}) events.APIGatewayProxyResponse {
	body := ErrorBody{
		Error:   errorCode,
		Message: message,
		Details: details,
	}

	bodyJSON, _ := json.Marshal(body)

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    defaultHeaders,
		Body:       string(bodyJSON),
	}
}

// SuccessResponse creates an API Gateway success response
func SuccessResponse(data interface{}, statusCode int) events.APIGatewayProxyResponse {
	bodyJSON, _ := json.Marshal(data)

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    defaultHeaders,
		Body:       string(bodyJSON),
	}
}

// StepFunctionResponse represents a response for Step Functions
type StepFunctionResponse struct {
	StatusCode int    `json:"statusCode"`
	PaymentID  string `json:"paymentId"`
	Status     string `json:"status,omitempty"`
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
	CreatedAt  string `json:"createdAt,omitempty"`
}
