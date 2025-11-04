# DynamoDB Tables
module "payments_table" {
  source      = "./dynamodb/payments"
  environment = var.environment
}

# Lambda Functions
module "create_payment_lambda" {
  source              = "./lambda/create_payment"
  environment         = var.environment
  payments_table_name = module.payments_table.table_name
  payments_table_arn  = module.payments_table.table_arn
}

# API Gateway
module "api_gateway" {
  source                           = "./api_gateway"
  environment                      = var.environment
  api_acm_arn                      = ""
  create_payment_lambda_name       = module.create_payment_lambda.lambda_function_name
  create_payment_lambda_invoke_arn = module.create_payment_lambda.lambda_invoke_arn
}
