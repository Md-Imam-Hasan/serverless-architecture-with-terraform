resource "aws_api_gateway_rest_api" "api_gateway" {
  name        = "${var.environment}_payment_service_api"
  description = "Payment Service API"

  endpoint_configuration {
    types = ["REGIONAL"]
  }
}

# Api gateway account cloudwatch log permission 
resource "aws_api_gateway_account" "payment_service" {
  cloudwatch_role_arn = aws_iam_role.cloudwatch_payment_service.arn
}

resource "aws_iam_role" "cloudwatch_payment_service" {
  name = "api_gateway_cloudwatch_global_payment_service"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "apigateway.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "cloudwatch_payment_service" {
  name = "default"
  role = aws_iam_role.cloudwatch_payment_service.id

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:DescribeLogGroups",
                "logs:DescribeLogStreams",
                "logs:PutLogEvents",
                "logs:GetLogEvents",
                "logs:FilterLogEvents"
            ],
            "Resource": "*"
        }
    ]
}
EOF
}

# Payment API Resources
module "payments" {
  source                           = "./api_gateway_resources/payments"
  api_gateway_id                   = aws_api_gateway_rest_api.api_gateway.id
  api_gateway_root_resource_id     = aws_api_gateway_rest_api.api_gateway.root_resource_id
  api_gateway_execution_arn        = aws_api_gateway_rest_api.api_gateway.execution_arn
  request_body_validator_id        = aws_api_gateway_request_validator.RequestBody.id
  create_payment_lambda_name       = var.create_payment_lambda_name
  create_payment_lambda_invoke_arn = var.create_payment_lambda_invoke_arn
  depends_on                       = [aws_api_gateway_rest_api.api_gateway]
}

resource "aws_api_gateway_deployment" "api_gateway_deployment" {
  rest_api_id = aws_api_gateway_rest_api.api_gateway.id

  depends_on = [
    module.payments,
  ]
  
  triggers = {
    redeployment = sha1(jsonencode([
      timestamp()
    ]))

  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_api_gateway_stage" "api_gateway_stage" {
  deployment_id         = aws_api_gateway_deployment.api_gateway_deployment.id
  rest_api_id           = aws_api_gateway_rest_api.api_gateway.id
  stage_name            = var.environment
  # documentation_version = aws_api_gateway_documentation_version.Documentation.version

  depends_on = [aws_api_gateway_deployment.api_gateway_deployment]
}

resource "aws_api_gateway_method_settings" "all" {
  rest_api_id = aws_api_gateway_rest_api.api_gateway.id
  stage_name  = var.environment
  method_path = "*/*"

  settings {
    logging_level      = "INFO"
    metrics_enabled    = true
    data_trace_enabled = false
    # Limit the rate of calls to prevent abuse and unwanted charges
    throttling_rate_limit  = 100
    throttling_burst_limit = 50
  }

  depends_on = [
    aws_api_gateway_stage.api_gateway_stage,
    aws_api_gateway_deployment.api_gateway_deployment,
    aws_api_gateway_rest_api.api_gateway
  ]
}

resource "aws_api_gateway_usage_plan" "payment_service" {
  name = "payment_service_usage_plan"

  api_stages {
    api_id = aws_api_gateway_rest_api.api_gateway.id
    stage  = aws_api_gateway_stage.api_gateway_stage.stage_name
  }
  throttle_settings {
    burst_limit = 50
    rate_limit  = 100
  }

  depends_on = [
    aws_api_gateway_stage.api_gateway_stage,
    aws_api_gateway_deployment.api_gateway_deployment,
    aws_api_gateway_rest_api.api_gateway
  ]
}

# resource "aws_api_gateway_usage_plan_key" "payment_service" {
#   key_id        = aws_api_gateway_api_key.payment_service.id
#   key_type      = "API_KEY"
#   usage_plan_id = aws_api_gateway_usage_plan.payment_service.id
#   depends_on = [
#     aws_api_gateway_stage.api_gateway_stage,
#     aws_api_gateway_deployment.api_gateway_deployment,
#     aws_api_gateway_rest_api.api_gateway,
#     aws_api_gateway_usage_plan.payment_service
#   ]
# }

# Custom Domain Name
# resource "aws_api_gateway_domain_name" "api_gateway" {

#   domain_name              = var.environment == "prod" ? "api.payment-service.net" : "api.${var.environment}.payment-service.net"
#   regional_certificate_arn = var.api_acm_arn

#   security_policy = "TLS_1_2"
#   endpoint_configuration {
#     types = ["REGIONAL"]
#   }
# }

# resource "aws_api_gateway_base_path_mapping" "api_gateway" {
#   api_id      = aws_api_gateway_rest_api.api_gateway.id
#   stage_name  = aws_api_gateway_stage.api_gateway_stage.stage_name
#   domain_name = aws_api_gateway_domain_name.api_gateway.domain_name

# }