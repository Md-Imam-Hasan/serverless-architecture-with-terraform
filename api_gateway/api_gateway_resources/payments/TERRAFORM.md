# payments

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
| <a name="module_cors"></a> [cors](#module\_cors) | squidfunk/api-gateway-enable-cors/aws | n/a |

## Resources

| Name | Type |
|------|------|
| [aws_api_gateway_integration.payments_post_integration](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_integration) | resource |
| [aws_api_gateway_method.payments_post_method](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_method) | resource |
| [aws_api_gateway_resource.payments_resource](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/api_gateway_resource) | resource |
| [aws_lambda_permission.payments_post_permission](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/lambda_permission) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_api_gateway_execution_arn"></a> [api\_gateway\_execution\_arn](#input\_api\_gateway\_execution\_arn) | n/a | `string` | n/a | yes |
| <a name="input_api_gateway_id"></a> [api\_gateway\_id](#input\_api\_gateway\_id) | n/a | `string` | n/a | yes |
| <a name="input_api_gateway_root_resource_id"></a> [api\_gateway\_root\_resource\_id](#input\_api\_gateway\_root\_resource\_id) | n/a | `string` | n/a | yes |
| <a name="input_create_payment_lambda_invoke_arn"></a> [create\_payment\_lambda\_invoke\_arn](#input\_create\_payment\_lambda\_invoke\_arn) | n/a | `string` | n/a | yes |
| <a name="input_create_payment_lambda_name"></a> [create\_payment\_lambda\_name](#input\_create\_payment\_lambda\_name) | n/a | `string` | n/a | yes |
| <a name="input_request_body_validator_id"></a> [request\_body\_validator\_id](#input\_request\_body\_validator\_id) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_payments_post_integration_id"></a> [payments\_post\_integration\_id](#output\_payments\_post\_integration\_id) | n/a |
| <a name="output_payments_post_method_id"></a> [payments\_post\_method\_id](#output\_payments\_post\_method\_id) | n/a |
| <a name="output_payments_resource_id"></a> [payments\_resource\_id](#output\_payments\_resource\_id) | n/a |
<!-- END OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
