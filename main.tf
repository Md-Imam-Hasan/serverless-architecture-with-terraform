# S3 Buckets
module "lambda_artifacts_bucket" {
  source      = "./s3/lambda_artifacts"
  environment = var.environment
  account_id  = var.account[var.environment]
}

# DynamoDB Tables
module "payments_table" {
  source      = "./dynamodb/payments"
  environment = var.environment
}

# SSM Parameter Store
module "ssm" {
  source      = "./ssm"
  environment = var.environment
  region      = var.region
  account_id  = var.account[var.environment]
}

# Lambda Functions
module "create_payment_lambda" {
  source                  = "./lambda/create_payment"
  environment             = var.environment
  region                  = var.region
  account_id              = var.account[var.environment]
  payments_table_name     = module.payments_table.table_name
  payments_table_arn      = module.payments_table.table_arn
  lambda_artifacts_bucket = module.lambda_artifacts_bucket.bucket_name
  lambda_artifacts_key    = var.lambda_artifacts_key
  lambda_source_code_hash = var.lambda_source_code_hash
  ssm_read_policy_arn     = module.ssm.ssm_read_policy_arn
  state_machine_arn       = module.step_functions.state_machine_arn

  depends_on = [module.ssm, module.step_functions]
}

module "step_handler_lambda" {
  source                  = "./lambda/step_handler"
  environment             = var.environment
  region                  = var.region
  account_id              = var.account[var.environment]
  payments_table_name     = module.payments_table.table_name
  payments_table_arn      = module.payments_table.table_arn
  lambda_artifacts_bucket = module.lambda_artifacts_bucket.bucket_name
  lambda_artifacts_key    = var.step_handler_lambda_artifacts_key
  lambda_source_code_hash = var.step_handler_lambda_source_code_hash
  ssm_read_policy_arn     = module.ssm.ssm_read_policy_arn
  event_bus_name          = var.event_bus_name

  depends_on = [module.ssm]
}

module "stripe_webhook_lambda" {
  source                  = "./lambda/stripe_webhook"
  environment             = var.environment
  region                  = var.region
  account_id              = var.account[var.environment]
  payments_table_name     = module.payments_table.table_name
  payments_table_arn      = module.payments_table.table_arn
  lambda_artifacts_bucket = module.lambda_artifacts_bucket.bucket_name
  lambda_artifacts_key    = var.stripe_webhook_lambda_artifacts_key
  lambda_source_code_hash = var.stripe_webhook_lambda_source_code_hash
  ssm_read_policy_arn     = module.ssm.ssm_read_policy_arn
  event_bus_name          = var.event_bus_name

  depends_on = [module.ssm]
}

# Step Functions
module "step_functions" {
  source                  = "./step_functions"
  environment             = var.environment
  step_handler_lambda_arn = module.step_handler_lambda.lambda_function_arn

  depends_on = [module.step_handler_lambda]
}

# API Gateway
module "api_gateway" {
  source                           = "./api_gateway"
  environment                      = var.environment
  api_acm_arn                      = ""
  create_payment_lambda_name       = module.create_payment_lambda.lambda_function_name
  create_payment_lambda_invoke_arn = module.create_payment_lambda.lambda_invoke_arn
  stripe_webhook_lambda_name       = module.stripe_webhook_lambda.lambda_function_name
  stripe_webhook_lambda_invoke_arn = module.stripe_webhook_lambda.lambda_invoke_arn
}
