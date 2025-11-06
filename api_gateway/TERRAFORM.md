# api_gateway

<!-- BEGINNING OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 6.19.0 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_payments"></a> [payments](#module\_payments) | ./api_gateway_resources/payments | n/a |

## Resources

| Name | Type |
|------|------|
| [aws_api_gateway_account.payment_service](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_account) | resource |
| [aws_api_gateway_api_key.payment_service](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_api_key) | resource |
| [aws_api_gateway_deployment.api_gateway_deployment](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_deployment) | resource |
| [aws_api_gateway_method_settings.all](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_method_settings) | resource |
| [aws_api_gateway_request_validator.RequestBody](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_request_validator) | resource |
| [aws_api_gateway_request_validator.RequestParameter](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_request_validator) | resource |
| [aws_api_gateway_rest_api.api_gateway](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_rest_api) | resource |
| [aws_api_gateway_stage.api_gateway_stage](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_stage) | resource |
| [aws_api_gateway_usage_plan.payment_service](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_usage_plan) | resource |
| [aws_api_gateway_usage_plan_key.payment_service](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_usage_plan_key) | resource |
| [aws_iam_role.cloudwatch_payment_service](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role_policy.cloudwatch_payment_service](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_api_acm_arn"></a> [api\_acm\_arn](#input\_api\_acm\_arn) | n/a | `string` | n/a | yes |
| <a name="input_create_payment_lambda_invoke_arn"></a> [create\_payment\_lambda\_invoke\_arn](#input\_create\_payment\_lambda\_invoke\_arn) | n/a | `string` | n/a | yes |
| <a name="input_create_payment_lambda_name"></a> [create\_payment\_lambda\_name](#input\_create\_payment\_lambda\_name) | n/a | `string` | n/a | yes |
| <a name="input_environment"></a> [environment](#input\_environment) | release environment name | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_api_gateway_api_key_id"></a> [api\_gateway\_api\_key\_id](#output\_api\_gateway\_api\_key\_id) | The ID of the API Gateway API key |
| <a name="output_api_gateway_api_key_value"></a> [api\_gateway\_api\_key\_value](#output\_api\_gateway\_api\_key\_value) | The value of the API Gateway API key |
| <a name="output_api_gateway_deployment_id"></a> [api\_gateway\_deployment\_id](#output\_api\_gateway\_deployment\_id) | The ID of the API Gateway deployment |
| <a name="output_api_gateway_execution_arn"></a> [api\_gateway\_execution\_arn](#output\_api\_gateway\_execution\_arn) | The execution ARN of the API Gateway |
| <a name="output_api_gateway_id"></a> [api\_gateway\_id](#output\_api\_gateway\_id) | The ID of the API Gateway REST API |
| <a name="output_api_gateway_invoke_url"></a> [api\_gateway\_invoke\_url](#output\_api\_gateway\_invoke\_url) | The invoke URL of the API Gateway stage |
| <a name="output_api_gateway_root_resource_id"></a> [api\_gateway\_root\_resource\_id](#output\_api\_gateway\_root\_resource\_id) | The root resource ID of the API Gateway |
| <a name="output_api_gateway_stage_name"></a> [api\_gateway\_stage\_name](#output\_api\_gateway\_stage\_name) | The name of the API Gateway stage |
| <a name="output_api_gateway_usage_plan_id"></a> [api\_gateway\_usage\_plan\_id](#output\_api\_gateway\_usage\_plan\_id) | The ID of the API Gateway usage plan |
| <a name="output_cloudwatch_role_arn"></a> [cloudwatch\_role\_arn](#output\_cloudwatch\_role\_arn) | The ARN of the CloudWatch IAM role for API Gateway |
| <a name="output_request_body_validator_id"></a> [request\_body\_validator\_id](#output\_request\_body\_validator\_id) | The ID of the request body validator |
| <a name="output_request_parameter_validator_id"></a> [request\_parameter\_validator\_id](#output\_request\_parameter\_validator\_id) | The ID of the request parameter validator |
<!-- END OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
