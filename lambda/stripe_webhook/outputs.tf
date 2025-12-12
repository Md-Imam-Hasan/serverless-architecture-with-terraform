output "lambda_function_name" {
  value = aws_lambda_function.stripe_webhook.function_name
}

output "lambda_function_arn" {
  value = aws_lambda_function.stripe_webhook.arn
}

output "lambda_invoke_arn" {
  value = aws_lambda_function.stripe_webhook.invoke_arn
}
