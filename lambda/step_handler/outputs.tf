output "lambda_function_name" {
  value = aws_lambda_function.step_handler.function_name
}

output "lambda_function_arn" {
  value = aws_lambda_function.step_handler.arn
}

output "lambda_invoke_arn" {
  value = aws_lambda_function.step_handler.invoke_arn
}
