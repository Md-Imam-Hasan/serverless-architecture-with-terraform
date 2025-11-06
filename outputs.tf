output "api_gateway_invoke_url" {
  description = "The invoke URL of the API Gateway stage"
  value       = module.api_gateway.api_gateway_invoke_url
}

output "api_gateway_api_key_value" {
  description = "The API Gateway API key value"
  value       = module.api_gateway.api_gateway_api_key_value
  sensitive   = true
}

output "api_gateway_api_key_id" {
  description = "The API Gateway API key ID"
  value       = module.api_gateway.api_gateway_api_key_id
}

output "create_payment_lambda_name" {
  description = "Create payment Lambda function name"
  value       = module.create_payment_lambda.lambda_function_name
}

output "create_payment_lambda_arn" {
  description = "Create payment Lambda function ARN"
  value       = module.create_payment_lambda.lambda_function_arn
}
