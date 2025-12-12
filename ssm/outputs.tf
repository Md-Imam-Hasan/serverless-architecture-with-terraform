output "ssm_read_policy_arn" {
  description = "ARN of the IAM policy for reading SSM parameters"
  value       = aws_iam_policy.ssm_read_policy.arn
}

output "ssm_parameter_prefix" {
  description = "Prefix for SSM parameters"
  value       = "/payment-service/${var.environment}"
}
