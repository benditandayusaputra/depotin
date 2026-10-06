package order

const (
	StatusPending    = "pending"
	StatusConfirmed  = "confirmed"
	StatusOnDelivery = "on_delivery"
	StatusDelivered  = "delivered"
	StatusCancelled  = "cancelled"

	SourcePublic   = "public"
	SourceLink     = "link"
	SourceReminder = "reminder"
	SourceOwner    = "owner"
	SourceCourier  = "courier"

	FulfilmentDelivery = "delivery"
	FulfilmentPickup   = "pickup"

	PaymentCash     = "cash"
	PaymentTransfer = "transfer"
	PaymentQRIS     = "qris"
	PaymentUnpaid   = "unpaid"
	PaymentPaid     = "paid"

	EventCreated    = "created"
	EventConfirmed  = "confirmed"
	EventAssigned   = "assigned"
	EventDispatched = "dispatched"
	EventDelivered  = "delivered"
	EventCancelled  = "cancelled"
	EventPaid       = "paid"
)

var transitions = map[string]map[string]bool{
	StatusPending:    {StatusConfirmed: true, StatusCancelled: true},
	StatusConfirmed:  {StatusOnDelivery: true, StatusDelivered: true, StatusCancelled: true},
	StatusOnDelivery: {StatusDelivered: true, StatusCancelled: true},
	StatusDelivered:  {},
	StatusCancelled:  {},
}

func CanTransition(from, to string) bool {
	return transitions[from][to]
}

func IsActive(status string) bool {
	return status == StatusPending || status == StatusConfirmed || status == StatusOnDelivery
}

func IsFinal(status string) bool {
	return status == StatusDelivered || status == StatusCancelled
}

var ActiveStatuses = []string{StatusPending, StatusConfirmed, StatusOnDelivery}
