output "bucket_name" {
  value       = aws_s3_bucket.lambda_artifacts.bucket
  description = "Name of the Lambda artifacts S3 bucket"
}

output "bucket_arn" {
  value       = aws_s3_bucket.lambda_artifacts.arn
  description = "ARN of the Lambda artifacts S3 bucket"
}
