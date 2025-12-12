# Step Functions State Machine for Scheduled Payments
# Waits until scheduledAt time, then invokes step_handler Lambda

resource "aws_sfn_state_machine" "scheduled_payment" {
  name     = "${var.environment}_scheduled_payment_processor"
  role_arn = aws_iam_role.step_functions_role.arn

  definition = jsonencode({
    Comment = "Process scheduled payments - wait until scheduledAt then generate payment link"
    StartAt = "WaitUntilScheduledTime"
    States = {
      WaitUntilScheduledTime = {
        Type          = "Wait"
        TimestampPath = "$.scheduledAt"
        Next          = "GeneratePaymentLink"
      }
      GeneratePaymentLink = {
        Type     = "Task"
        Resource = "arn:aws:states:::lambda:invoke"
        Parameters = {
          FunctionName = var.step_handler_lambda_arn
          Payload = {
            "paymentId.$"   = "$.paymentId"
            "orderNumber.$" = "$.orderNumber"
            "companyName.$" = "$.companyName"
            "amount.$"      = "$.amount"
            "currency.$"    = "$.currency"
            "sourceKey.$"   = "$.sourceKey"
          }
        }
        OutputPath = "$.Payload"
        Retry = [
          {
            ErrorEquals = [
              "Lambda.ServiceException",
              "Lambda.AWSLambdaException",
              "Lambda.SdkClientException",
              "Lambda.TooManyRequestsException"
            ]
            IntervalSeconds = 2
            MaxAttempts     = 3
            BackoffRate     = 2
          }
        ]
        Catch = [
          {
            ErrorEquals = ["States.ALL"]
            Next        = "HandleError"
            ResultPath  = "$.error"
          }
        ]
        End = true
      }
      HandleError = {
        Type = "Pass"
        Parameters = {
          "status"      = "error"
          "message"     = "Failed to generate payment link"
          "error.$"     = "$.error"
          "paymentId.$" = "$.paymentId"
        }
        End = true
      }
    }
  })

  logging_configuration {
    log_destination        = "${aws_cloudwatch_log_group.step_functions_logs.arn}:*"
    include_execution_data = true
    level                  = "ALL"
  }

  tags = {
    Name        = "${var.environment}_scheduled_payment_processor"
    Environment = var.environment
  }
}

# IAM Role for Step Functions
resource "aws_iam_role" "step_functions_role" {
  name = "${var.environment}_step_functions_scheduled_payment_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "states.amazonaws.com"
        }
      }
    ]
  })
}

# IAM Policy for Step Functions to invoke Lambda
resource "aws_iam_role_policy" "step_functions_lambda_policy" {
  name = "${var.environment}_step_functions_lambda_policy"
  role = aws_iam_role.step_functions_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "lambda:InvokeFunction"
        ]
        Resource = [
          var.step_handler_lambda_arn,
          "${var.step_handler_lambda_arn}:*"
        ]
      }
    ]
  })
}

# IAM Policy for Step Functions CloudWatch Logs
resource "aws_iam_role_policy" "step_functions_logs_policy" {
  name = "${var.environment}_step_functions_logs_policy"
  role = aws_iam_role.step_functions_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogDelivery",
          "logs:GetLogDelivery",
          "logs:UpdateLogDelivery",
          "logs:DeleteLogDelivery",
          "logs:ListLogDeliveries",
          "logs:PutLogEvents",
          "logs:PutResourcePolicy",
          "logs:DescribeResourcePolicies",
          "logs:DescribeLogGroups"
        ]
        Resource = "*"
      }
    ]
  })
}

# CloudWatch Log Group for Step Functions
resource "aws_cloudwatch_log_group" "step_functions_logs" {
  name              = "/aws/vendedlogs/states/${var.environment}_scheduled_payment_processor"
  retention_in_days = 7

  tags = {
    Name        = "${var.environment}_scheduled_payment_processor_logs"
    Environment = var.environment
  }
}
