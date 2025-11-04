import json
import logging
import os
from datetime import datetime
from uuid import uuid4

import boto3

logger = logging.getLogger()
logger.setLevel(logging.INFO)

dynamodb = boto3.resource("dynamodb")
table = dynamodb.Table(os.environ["PAYMENTS_TABLE_NAME"])


def lambda_handler(event, context):
    """
    Create payment Lambda function
    """
    trace_id = context.request_id
    logger.info(json.dumps({"event": event, "traceId": trace_id}))

    try:
        # Parse request body
        body = json.loads(event.get("body", "{}"))

        # Generate payment ID
        payment_id = str(uuid4())
        timestamp = datetime.utcnow().isoformat() + "Z"

        # Build payment item
        payment_item = {
            "PK": f"PAYMENT#{payment_id}",
            "SK": "METADATA",
            "paymentId": payment_id,
            "orderNumber": body["orderNumber"],
            "companyId": body["companyId"],
            "status": "created",
            "paymentMode": body["paymentMode"],
            "paymentType": body["paymentType"],
            "provider": body["provider"],
            "amount": body["amount"],
            "currency": body["currency"],
            "sourceKey": body["sourceKey"],
            "createdAt": timestamp,
            "updatedAt": timestamp,
        }

        # Add optional fields
        optional_fields = [
            "providerPaymentId",
            "paymentLinkId",
            "paymentLink",
            "cancelUrl",
            "successUrl",
            "receiveUrl",
            "tripId",
            "scheduledAt",
            "expiresAt",
            "paidAt",
            "retryAfterTimestamp",
            "callbackUrl",
        ]
        for field in optional_fields:
            if field in body:
                payment_item[field] = body[field]

        # Write to DynamoDB
        table.put_item(Item=payment_item)

        logger.info(
            json.dumps(
                {
                    "message": "Payment created",
                    "paymentId": payment_id,
                    "traceId": trace_id,
                }
            )
        )

        return {
            "statusCode": 201,
            "headers": {
                "Content-Type": "application/json",
                "Access-Control-Allow-Origin": "*",
                "Access-Control-Allow-Headers": "Content-Type,Authorization",
                "Access-Control-Allow-Methods": "OPTIONS,POST,GET",
            },
            "body": json.dumps(
                {
                    "paymentId": payment_id,
                    "status": "created",
                    "createdAt": timestamp,
                    "traceId": trace_id,
                }
            ),
        }

    except Exception as e:
        logger.error(json.dumps({"error": str(e), "traceId": trace_id}))
        return {
            "statusCode": 500,
            "headers": {
                "Content-Type": "application/json",
                "Access-Control-Allow-Origin": "*",
            },
            "body": json.dumps({"error": "Internal server error", "traceId": trace_id}),
        }
