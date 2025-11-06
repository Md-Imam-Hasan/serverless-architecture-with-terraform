# POST /payments - Create payment & payment link
resource "aws_api_gateway_resource" "payments_resource" {
  rest_api_id = var.api_gateway_id
  parent_id   = var.api_gateway_root_resource_id
  path_part   = "payments"
}

resource "aws_api_gateway_method" "payments_post_method" {
  rest_api_id          = var.api_gateway_id
  resource_id          = aws_api_gateway_resource.payments_resource.id
  http_method          = "POST"
  authorization        = "NONE"
  api_key_required     = false
  request_validator_id = var.request_body_validator_id

  depends_on = [aws_api_gateway_resource.payments_resource]
}

resource "aws_lambda_permission" "payments_post_permission" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = var.create_payment_lambda_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${var.api_gateway_execution_arn}/*/*"

  depends_on = [aws_api_gateway_method.payments_post_method]
}

resource "aws_api_gateway_integration" "payments_post_integration" {
  rest_api_id             = var.api_gateway_id
  resource_id             = aws_api_gateway_resource.payments_resource.id
  http_method             = aws_api_gateway_method.payments_post_method.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = var.create_payment_lambda_invoke_arn

  depends_on = [aws_api_gateway_method.payments_post_method, aws_lambda_permission.payments_post_permission]
}

module "cors" {
  source = "squidfunk/api-gateway-enable-cors/aws"

  api_id          = var.api_gateway_id
  api_resource_id = aws_api_gateway_resource.payments_resource.id
  allow_headers = [
    "Authorization",
    "Content-Type",
  ]
  allow_methods = [
    "OPTIONS",
    "HEAD",
    "GET",
    "POST",
    "PUT",
    "PATCH",
    "DELETE"
  ]
  allow_origin      = "*"
  allow_max_age     = "7200"
  allow_credentials = false
}
