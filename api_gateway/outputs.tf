output "api_gateway_id" {
  description = "The ID of the API Gateway REST API"
  value       = aws_api_gateway_rest_api.api_gateway.id
}

output "api_gateway_root_resource_id" {
  description = "The root resource ID of the API Gateway"
  value       = aws_api_gateway_rest_api.api_gateway.root_resource_id
}

output "api_gateway_execution_arn" {
  description = "The execution ARN of the API Gateway"
  value       = aws_api_gateway_rest_api.api_gateway.execution_arn
}

output "api_gateway_invoke_url" {
  description = "The invoke URL of the API Gateway stage"
  value       = aws_api_gateway_stage.api_gateway_stage.invoke_url
}

output "api_gateway_stage_name" {
  description = "The name of the API Gateway stage"
  value       = aws_api_gateway_stage.api_gateway_stage.stage_name
}

output "api_gateway_deployment_id" {
  description = "The ID of the API Gateway deployment"
  value       = aws_api_gateway_deployment.api_gateway_deployment.id
}

output "api_gateway_usage_plan_id" {
  description = "The ID of the API Gateway usage plan"
  value       = aws_api_gateway_usage_plan.payment_service.id
}

# output "api_gateway_api_key_id" {
#   description = "The ID of the API Gateway API key"
#   value       = aws_api_gateway_api_key.payment_service.id
# }

# output "api_gateway_domain_name" {
#   description = "The custom domain name for the API Gateway"
#   value       = aws_api_gateway_domain_name.api_gateway.domain_name
# }

output "cloudwatch_role_arn" {
  description = "The ARN of the CloudWatch IAM role for API Gateway"
  value       = aws_iam_role.cloudwatch_payment_service.arn
}

output "request_body_validator_id" {
  description = "The ID of the request body validator"
  value       = aws_api_gateway_request_validator.RequestBody.id
}

output "request_parameter_validator_id" {
  description = "The ID of the request parameter validator"
  value       = aws_api_gateway_request_validator.RequestParameter.id
}
