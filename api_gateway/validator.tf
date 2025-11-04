resource "aws_api_gateway_request_validator" "RequestBody" {
  name                        = "RequestBodyValidator"
  rest_api_id                 = aws_api_gateway_rest_api.api_gateway.id
  validate_request_body       = true
  validate_request_parameters = false
}
resource "aws_api_gateway_request_validator" "RequestParameter" {
  name                        = "RequestParameterValidator"
  rest_api_id                 = aws_api_gateway_rest_api.api_gateway.id
  validate_request_body       = false
  validate_request_parameters = true
}