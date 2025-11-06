# payment-service

<!-- BEGINNING OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.5.0 |
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | ~> 5.0 |

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_api_gateway"></a> [api\_gateway](#module\_api\_gateway) | ./api_gateway | n/a |
| <a name="module_create_payment_lambda"></a> [create\_payment\_lambda](#module\_create\_payment\_lambda) | ./lambda/create_payment | n/a |
| <a name="module_payments_table"></a> [payments\_table](#module\_payments\_table) | ./dynamodb/payments | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_account"></a> [account](#input\_account) | account ID for all environment | `map` | <pre>{<br>  "dev": "776724785158"<br>}</pre> | no |
| <a name="input_aws_role_arn"></a> [aws\_role\_arn](#input\_aws\_role\_arn) | Role to assume for deploying resources | `string` | n/a | yes |
| <a name="input_environment"></a> [environment](#input\_environment) | release environment name | `string` | `"dev"` | no |
| <a name="input_region"></a> [region](#input\_region) | region for all environments | `string` | `"ap-southeast-1"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_api_gateway_api_key_id"></a> [api\_gateway\_api\_key\_id](#output\_api\_gateway\_api\_key\_id) | The API Gateway API key ID |
| <a name="output_api_gateway_api_key_value"></a> [api\_gateway\_api\_key\_value](#output\_api\_gateway\_api\_key\_value) | The API Gateway API key value |
| <a name="output_api_gateway_invoke_url"></a> [api\_gateway\_invoke\_url](#output\_api\_gateway\_invoke\_url) | The invoke URL of the API Gateway stage |
| <a name="output_create_payment_lambda_arn"></a> [create\_payment\_lambda\_arn](#output\_create\_payment\_lambda\_arn) | Create payment Lambda function ARN |
| <a name="output_create_payment_lambda_name"></a> [create\_payment\_lambda\_name](#output\_create\_payment\_lambda\_name) | Create payment Lambda function name |
<!-- END OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
