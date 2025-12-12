# Step Handler Lambda - Processes scheduled payments from Step Functions

resource "aws_lambda_function" "step_handler" {
  s3_bucket        = var.lambda_artifacts_bucket
  s3_key           = var.lambda_artifacts_key
  function_name    = "${var.environment}_step_handler"
  role             = aws_iam_role.step_handler_lambda_role.arn
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
      EVENT_BUS_NAME      = var.event_bus_name
    }
  }

  tags = {
    Name        = "${var.environment}_step_handler"
    Environment = var.environment
  }
}

resource "aws_iam_role" "step_handler_lambda_role" {
  name = "${var.environment}_step_handler_lambda_role"

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

resource "aws_iam_role_policy_attachment" "step_handler_lambda_basic" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
  role       = aws_iam_role.step_handler_lambda_role.name
}

# DynamoDB policy
resource "aws_iam_role_policy" "step_handler_dynamodb_policy" {
  name = "${var.environment}_step_handler_dynamodb_policy"
  role = aws_iam_role.step_handler_lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:UpdateItem",
          "dynamodb:Query"
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
resource "aws_iam_role_policy_attachment" "step_handler_ssm_policy" {
  policy_arn = var.ssm_read_policy_arn
  role       = aws_iam_role.step_handler_lambda_role.name
}

# EventBridge policy
resource "aws_iam_role_policy" "step_handler_eventbridge_policy" {
  name = "${var.environment}_step_handler_eventbridge_policy"
  role = aws_iam_role.step_handler_lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "events:PutEvents"
        ]
        Resource = [
          "arn:aws:events:${var.region}:${var.account_id}:event-bus/${var.event_bus_name}"
        ]
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "step_handler_lambda_logs" {
  name              = "/aws/lambda/${aws_lambda_function.step_handler.function_name}"
  retention_in_days = 7
}
