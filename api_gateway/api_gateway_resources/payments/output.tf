output "payments_resource_id" {
  value = aws_api_gateway_resource.payments_resource.id
}
output "payments_post_method_id" {
  value = aws_api_gateway_method.payments_post_method.id
}
output "payments_post_integration_id" {
  value = aws_api_gateway_integration.payments_post_integration.id
}
