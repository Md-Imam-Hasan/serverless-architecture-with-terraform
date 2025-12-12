package enums

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusCreated              PaymentStatus = "created"
	PaymentStatusPendingLink          PaymentStatus = "pending_link"
	PaymentStatusLinkGenerated        PaymentStatus = "link_generated"
	PaymentStatusLinkGenerationFailed PaymentStatus = "link_generation_failed"
	PaymentStatusScheduled            PaymentStatus = "scheduled"
	PaymentStatusScheduledFailed      PaymentStatus = "scheduled_failed"
	PaymentStatusProcessing           PaymentStatus = "processing"
	PaymentStatusPaid                 PaymentStatus = "paid"
	PaymentStatusFailed               PaymentStatus = "failed"
	PaymentStatusCanceled             PaymentStatus = "canceled"
	PaymentStatusRefunded             PaymentStatus = "refunded"
)

func (s PaymentStatus) String() string {
	return string(s)
}

// IsValid checks if the status is a valid PaymentStatus
func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusCreated, PaymentStatusPendingLink, PaymentStatusLinkGenerated,
		PaymentStatusLinkGenerationFailed, PaymentStatusScheduled, PaymentStatusScheduledFailed,
		PaymentStatusProcessing, PaymentStatusPaid, PaymentStatusFailed,
		PaymentStatusCanceled, PaymentStatusRefunded:
		return true
	}
	return false
}

// PaymentMode represents the mode of payment
type PaymentMode string

const (
	PaymentModeOneTime      PaymentMode = "one_time"
	PaymentModeSubscription PaymentMode = "subscription"
)

func (m PaymentMode) String() string {
	return string(m)
}

// IsValid checks if the mode is a valid PaymentMode
func (m PaymentMode) IsValid() bool {
	switch m {
	case PaymentModeOneTime, PaymentModeSubscription:
		return true
	}
	return false
}

// ValidPaymentModes returns all valid payment modes
func ValidPaymentModes() []string {
	return []string{
		PaymentModeOneTime.String(),
		PaymentModeSubscription.String(),
	}
}

// PaymentType represents the type of payment
type PaymentType string

const (
	PaymentTypeCard         PaymentType = "card"
	PaymentTypeBankTransfer PaymentType = "bank_transfer"
	PaymentTypeWallet       PaymentType = "wallet"
)

func (t PaymentType) String() string {
	return string(t)
}

// IsValid checks if the type is a valid PaymentType
func (t PaymentType) IsValid() bool {
	switch t {
	case PaymentTypeCard, PaymentTypeBankTransfer, PaymentTypeWallet:
		return true
	}
	return false
}

// ValidPaymentTypes returns all valid payment types
func ValidPaymentTypes() []string {
	return []string{
		PaymentTypeCard.String(),
		PaymentTypeBankTransfer.String(),
		PaymentTypeWallet.String(),
	}
}

// EventType represents EventBridge event types
type EventType string

const (
	EventTypePaymentLinkCreated          EventType = "payment_link.created"
	EventTypePaymentLinkGenerationFailed EventType = "payment_link.generation_failed"
	EventTypePaymentIntentSucceeded      EventType = "payment_intent.succeeded"
	EventTypePaymentIntentFailed         EventType = "payment_intent.payment_failed"
	EventTypePaymentIntentCanceled       EventType = "payment_intent.canceled"
	EventTypePaymentIntentProcessing     EventType = "payment_intent.processing"
	EventTypeChargeRefunded              EventType = "charge.refunded"
)

func (e EventType) String() string {
	return string(e)
}
