# Create Payment Lambda

Lambda function that creates a new payment record in DynamoDB.

## Request Format

**POST** `/payments`

```json
{
  "orderNumber": "TT-9874-S01",
  "companyId": "comp_teq_001",
  "paymentMode": "single",
  "paymentType": "full_prepay",
  "provider": "stripe",
  "amount": 15000,
  "currency": "NOK",
  "sourceKey": "TEQ",
  "cancelUrl": "https://teq.example.com/payment/canceled",
  "successUrl": "https://teq.example.com/payment/success",
  "callbackUrl": "https://teq.example.com/api/payment-callback"
}
```

### Required Fields
- `orderNumber` - TEQ order identifier
- `companyId` - TEQ company/instance identifier
- `paymentMode` - Mode of payment (single, individual, batch)
- `paymentType` - Payment type (full_prepay, deposit, staged, post_trip)
- `provider` - Payment provider (stripe, vipps, klarna)
- `amount` - Total amount in smallest currency unit
- `currency` - ISO 4217 currency code (NOK, USD, EUR)
- `sourceKey` - Source system (TEQ, BusNetwork)

### Optional Fields
- `providerPaymentId` - Provider's payment/intent ID
- `paymentLinkId` - Provider's payment link/session ID
- `paymentLink` - Generated payment link URL
- `cancelUrl` - Redirect URL if customer cancels
- `successUrl` - Redirect URL after success
- `receiveUrl` - POST URL for staged payment link
- `tripId` - Associated trip identifier
- `scheduledAt` - When payment should be initiated (ISO8601)
- `expiresAt` - When payment link expires (ISO8601)
- `callbackUrl` - TEQ endpoint for status changes

## Response Format

**Success (201)**
```json
{
  "paymentId": "abc123def456",
  "status": "created",
  "createdAt": "2025-11-04T09:27:00.000000Z",
  "traceId": "request-id-here"
}
```

**Error (500)**
```json
{
  "error": "Internal server error",
  "traceId": "request-id-here"
}
```

## DynamoDB Schema

Stores payment with:
- **PK**: `PAYMENT#{paymentId}`
- **SK**: `METADATA`
- **GSI**: `CompanyPaymentsIndex` on `companyId` + `createdAt`

## Environment Variables

- `ENVIRONMENT` - Environment name (dev/test/staging/prod)
- `LOG_LEVEL` - Logging level (default: INFO)
- `PAYMENTS_TABLE_NAME` - DynamoDB table name

## Logging

Structured JSON logs with `traceId` for request tracking.
