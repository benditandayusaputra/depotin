package depot

import (
	"github.com/google/uuid"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
)

type View struct {
	ID                   uuid.UUID `json:"id"`
	Name                 string    `json:"name"`
	Slug                 string    `json:"slug"`
	Phone                string    `json:"phone"`
	Address              string    `json:"address"`
	Lat                  *float64  `json:"lat"`
	Lng                  *float64  `json:"lng"`
	Timezone             string    `json:"timezone"`
	OpenTime             string    `json:"open_time"`
	CloseTime            string    `json:"close_time"`
	DeliveryFee          int64     `json:"delivery_fee"`
	IsAcceptingOrders    bool      `json:"is_accepting_orders"`
	AutoConfirmKnown     bool      `json:"auto_confirm_known"`
	LoyaltyEvery         *int32    `json:"loyalty_every"`
	ReminderLeadDays     int32     `json:"reminder_lead_days"`
	DefaultDaysPerGallon float64   `json:"default_days_per_gallon"`
	IsDemo               bool      `json:"is_demo"`
}

func ToView(d sqlc.Depot) View {
	return View{
		ID: d.ID, Name: d.Name, Slug: d.Slug, Phone: d.Phone, Address: d.Address,
		Lat: d.Lat, Lng: d.Lng, Timezone: d.Timezone,
		OpenTime: shortTime(d.OpenTime), CloseTime: shortTime(d.CloseTime),
		DeliveryFee: d.DeliveryFee, IsAcceptingOrders: d.IsAcceptingOrders, AutoConfirmKnown: d.AutoConfirmKnown,
		LoyaltyEvery: d.LoyaltyEvery, ReminderLeadDays: d.ReminderLeadDays,
		DefaultDaysPerGallon: d.DefaultDaysPerGallon, IsDemo: d.IsDemo,
	}
}

func shortTime(t string) string {
	if len(t) >= 5 {
		return t[:5]
	}
	return t
}
