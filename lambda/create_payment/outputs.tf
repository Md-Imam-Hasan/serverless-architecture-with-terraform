output "lambda_function_name" {
  value = aws_lambda_function.create_payment.function_name
}

output "lambda_function_arn" {
  value = aws_lambda_function.create_payment.arn
}

output "lambda_invoke_arn" {
  value = aws_lambda_function.create_payment.invoke_arn
}
