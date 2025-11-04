output "table_name" {
  description = "Name of the DynamoDB payments table"
  value       = aws_dynamodb_table.payments.name
}

output "table_arn" {
  description = "ARN of the DynamoDB payments table"
  value       = aws_dynamodb_table.payments.arn
}

output "table_stream_arn" {
  description = "ARN of the DynamoDB table stream"
  value       = aws_dynamodb_table.payments.stream_arn
}
