# The Go Lambda binary is built by GitHub Actions before Terraform runs
# See: .github/workflows/deploy-dev.yml

data "archive_file" "create_payment_lambda_zip" {
  type        = "zip"
  source_file = "${path.module}/src/bootstrap"
  output_path = "${path.module}/lambda_function.zip"
}

resource "aws_lambda_function" "create_payment" {
  filename         = data.archive_file.create_payment_lambda_zip.output_path
  function_name    = "${var.environment}_create_payment"
  role             = aws_iam_role.create_payment_lambda_role.arn
  handler          = "bootstrap"
  source_code_hash = fileexists("${path.module}/src/bootstrap") ? filesha256("${path.module}/src/bootstrap") : filesha256("${path.module}/lambda_function.zip")
  runtime          = "provided.al2023"
  timeout          = 30
  memory_size      = 256
  architectures    = ["arm64"]

  environment {
    variables = {
      ENVIRONMENT         = var.environment
      LOG_LEVEL           = "INFO"
      PAYMENTS_TABLE_NAME = var.payments_table_name
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
          "dynamodb:Scan"
        ]
        Resource = [
          var.payments_table_arn,
          "${var.payments_table_arn}/index/*"
        ]
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "create_payment_lambda_logs" {
  name              = "/aws/lambda/${aws_lambda_function.create_payment.function_name}"
  retention_in_days = 7
}
