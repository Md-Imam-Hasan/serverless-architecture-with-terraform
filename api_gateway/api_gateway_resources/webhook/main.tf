# POST /webhook/stripe - Stripe webhook endpoint (public, no API key required)
resource "aws_api_gateway_resource" "webhook_resource" {
  rest_api_id = var.api_gateway_id
  parent_id   = var.api_gateway_root_resource_id
  path_part   = "webhook"
}

resource "aws_api_gateway_resource" "webhook_stripe_resource" {
  rest_api_id = var.api_gateway_id
  parent_id   = aws_api_gateway_resource.webhook_resource.id
  path_part   = "stripe"
}

resource "aws_api_gateway_method" "webhook_stripe_post_method" {
  rest_api_id      = var.api_gateway_id
  resource_id      = aws_api_gateway_resource.webhook_stripe_resource.id
  http_method      = "POST"
  authorization    = "NONE"
  api_key_required = false

  depends_on = [aws_api_gateway_resource.webhook_stripe_resource]
}

resource "aws_lambda_permission" "webhook_stripe_permission" {
  statement_id  = "AllowAPIGatewayInvokeStripeWebhook"
  action        = "lambda:InvokeFunction"
  function_name = var.stripe_webhook_lambda_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${var.api_gateway_execution_arn}/*/*"

  depends_on = [aws_api_gateway_method.webhook_stripe_post_method]
}

resource "aws_api_gateway_integration" "webhook_stripe_integration" {
  rest_api_id             = var.api_gateway_id
  resource_id             = aws_api_gateway_resource.webhook_stripe_resource.id
  http_method             = aws_api_gateway_method.webhook_stripe_post_method.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = var.stripe_webhook_lambda_invoke_arn

  depends_on = [aws_api_gateway_method.webhook_stripe_post_method, aws_lambda_permission.webhook_stripe_permission]
}
