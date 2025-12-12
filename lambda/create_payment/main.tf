# The Go Lambda binary is built by GitHub Actions and uploaded to S3
# See: .github/workflows/deploy-dev.yml

resource "aws_lambda_function" "create_payment" {
  s3_bucket        = var.lambda_artifacts_bucket
  s3_key           = var.lambda_artifacts_key
  function_name    = "${var.environment}_create_payment"
  role             = aws_iam_role.create_payment_lambda_role.arn
  handler          = "bootstrap"
  source_code_hash = var.lambda_source_code_hash
  runtime          = "provided.al2023"
  timeout          = 60
  memory_size      = 256
  architectures    = ["arm64"]

  environment {
    variables = {
      ENVIRONMENT         = var.environment
      LOG_LEVEL           = "INFO"
      PAYMENTS_TABLE_NAME = var.payments_table_name
      STATE_MACHINE_ARN   = var.state_machine_arn
    }
  }

  tags = {
    Name        = "${var.environment}_create_payment"
    Environment = var.environment
  }
}

resource "aws_iam_role" "create_payment_lambda_role" {
  name = "${var.environment}_create_payment_lambda_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "create_payment_lambda_basic" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
  role       = aws_iam_role.create_payment_lambda_role.name
}

resource "aws_iam_role_policy" "create_payment_dynamodb_policy" {
  name = "${var.environment}_create_payment_dynamodb_policy"
  role = aws_iam_role.create_payment_lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:UpdateItem",
          "dynamodb:Query",
          "dynamodb:Scan",
          "dynamodb:TransactWriteItems"
        ]
        Resource = [
          var.payments_table_arn,
          "${var.payments_table_arn}/index/*"
        ]
      }
    ]
  })
}

# SSM Parameter Store policy
resource "aws_iam_role_policy_attachment" "create_payment_ssm_policy" {
  policy_arn = var.ssm_read_policy_arn
  role       = aws_iam_role.create_payment_lambda_role.name
}

# Step Functions policy
resource "aws_iam_role_policy" "create_payment_step_functions_policy" {
  name = "${var.environment}_create_payment_step_functions_policy"
  role = aws_iam_role.create_payment_lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "states:StartExecution"
        ]
        Resource = [
          var.state_machine_arn
        ]
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "create_payment_lambda_logs" {
  name              = "/aws/lambda/${aws_lambda_function.create_payment.function_name}"
  retention_in_days = 7
}
