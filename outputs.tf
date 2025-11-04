output "api_gateway_invoke_url" {
  description = "API Gateway invoke URL"
  value       = module.api_gateway.api_gateway_invoke_url
}

output "create_payment_lambda_name" {
  description = "Create payment Lambda function name"
  value       = module.create_payment_lambda.lambda_function_name
}

output "create_payment_lambda_arn" {
  description = "Create payment Lambda function ARN"
  value       = module.create_payment_lambda.lambda_function_arn
}
