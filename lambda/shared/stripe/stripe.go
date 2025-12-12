package stripe

import (
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentlink"
	"github.com/stripe/stripe-go/v76/price"
	"github.com/stripe/stripe-go/v76/product"
)

// PaymentLinkResult contains the result of creating a Stripe payment link
type PaymentLinkResult struct {
	PaymentLink       string `json:"payment_link"`
	PaymentLinkID     string `json:"payment_link_id"`
	ProviderPaymentID string `json:"provider_payment_id"`
	CreatedAt         string `json:"created_at"`
}

// CreatePaymentLinkParams contains parameters for creating a payment link
type CreatePaymentLinkParams struct {
	PaymentID          string
	CompanyName        string
	Amount             int // Amount in cents
	Currency           string
	Description        string
	StripeSecretKey    string
	SourceKey          string
	RedirectURL        string
	AdditionalMetadata map[string]string
}

// CreatePaymentLink creates a Stripe payment link
func CreatePaymentLink(params *CreatePaymentLinkParams) (*PaymentLinkResult, error) {
	// Set the API key for this request
	stripe.Key = params.StripeSecretKey

	// Create a product
	productParams := &stripe.ProductParams{
		Name: stripe.String(params.Description),
	}
	productParams.AddMetadata("paymentId", params.PaymentID)
	productParams.AddMetadata("companyName", params.CompanyName)
	productParams.AddMetadata("sourceKey", params.SourceKey)
	for k, v := range params.AdditionalMetadata {
		productParams.AddMetadata(k, v)
	}

	prod, err := product.New(productParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Create a price
	priceParams := &stripe.PriceParams{
		Product:    stripe.String(prod.ID),
		UnitAmount: stripe.Int64(int64(params.Amount)),
		Currency:   stripe.String(params.Currency),
	}

	pr, err := price.New(priceParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create price: %w", err)
	}

	// Create payment link
	linkParams := &stripe.PaymentLinkParams{
		LineItems: []*stripe.PaymentLinkLineItemParams{
			{
				Price:    stripe.String(pr.ID),
				Quantity: stripe.Int64(1),
			},
		},
	}

	// Add redirect URL if provided
	if params.RedirectURL != "" {
		linkParams.AfterCompletion = &stripe.PaymentLinkAfterCompletionParams{
			Type: stripe.String("redirect"),
			Redirect: &stripe.PaymentLinkAfterCompletionRedirectParams{
				URL: stripe.String(params.RedirectURL),
			},
		}
	}

	// Add metadata to payment link
	linkParams.AddMetadata("paymentId", params.PaymentID)
	linkParams.AddMetadata("companyName", params.CompanyName)
	linkParams.AddMetadata("sourceKey", params.SourceKey)
	for k, v := range params.AdditionalMetadata {
		linkParams.AddMetadata(k, v)
	}

	// Transfer metadata to payment intent
	linkParams.PaymentIntentData = &stripe.PaymentLinkPaymentIntentDataParams{}
	linkParams.PaymentIntentData.AddMetadata("paymentId", params.PaymentID)
	linkParams.PaymentIntentData.AddMetadata("companyName", params.CompanyName)
	linkParams.PaymentIntentData.AddMetadata("sourceKey", params.SourceKey)
	for k, v := range params.AdditionalMetadata {
		linkParams.PaymentIntentData.AddMetadata(k, v)
	}

	link, err := paymentlink.New(linkParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment link: %w", err)
	}

	return &PaymentLinkResult{
		PaymentLink:       link.URL,
		PaymentLinkID:     link.ID,
		ProviderPaymentID: prod.ID,
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// DeactivatePaymentLink deactivates a Stripe payment link
func DeactivatePaymentLink(paymentLinkID, stripeSecretKey string) error {
	stripe.Key = stripeSecretKey

	_, err := paymentlink.Update(paymentLinkID, &stripe.PaymentLinkParams{
		Active: stripe.Bool(false),
	})
	if err != nil {
		return fmt.Errorf("failed to deactivate payment link: %w", err)
	}

	return nil
}
