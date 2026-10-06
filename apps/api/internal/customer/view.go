package customer

import (
	"time"

	"github.com/google/uuid"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
)

type View struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	Phone                string     `json:"phone"`
	Address              string     `json:"address"`
	AddressNote          string     `json:"address_note"`
	Area                 string     `json:"area"`
	Lat                  *float64   `json:"lat"`
	Lng                  *float64   `json:"lng"`
	Source               string     `json:"source"`
	IsVerified           bool       `json:"is_verified"`
	UsualQty             int32      `json:"usual_qty"`
	LoanBalance          int32      `json:"loan_balance"`
	StampCount           int32      `json:"stamp_count"`
	DaysPerGallon        *float64   `json:"days_per_gallon"`
	PredictionSamples    int32      `json:"prediction_samples"`
	PredictionConfidence string     `json:"prediction_confidence"`
	LastDeliveredAt      *time.Time `json:"last_delivered_at"`
	LastDeliveredQty     *int32     `json:"last_delivered_qty"`
	PredictedEmptyAt     *time.Time `json:"predicted_empty_at"`
	ReminderSnoozedUntil *string    `json:"reminder_snoozed_until"`
	IsActive             bool       `json:"is_active"`
	HasLink              bool       `json:"has_link"`
	CreatedAt            time.Time  `json:"created_at"`
}

func ToView(c sqlc.Customer) View {
	var snoozed *string
	if c.ReminderSnoozedUntil != nil {
		s := c.ReminderSnoozedUntil.Format(time.DateOnly)
		snoozed = &s
	}
	return View{
		ID: c.ID, Name: c.Name, Phone: c.Phone, Address: c.Address, AddressNote: c.AddressNote, Area: c.Area,
		Lat: c.Lat, Lng: c.Lng, Source: c.Source, IsVerified: c.IsVerified, UsualQty: c.UsualQty,
		LoanBalance: c.LoanBalance, StampCount: c.StampCount, DaysPerGallon: c.DaysPerGallon,
		PredictionSamples: c.PredictionSamples, PredictionConfidence: c.PredictionConfidence,
		LastDeliveredAt: c.LastDeliveredAt, LastDeliveredQty: c.LastDeliveredQty, PredictedEmptyAt: c.PredictedEmptyAt,
		ReminderSnoozedUntil: snoozed, IsActive: c.IsActive, HasLink: len(c.TokenHash) > 0, CreatedAt: c.CreatedAt,
	}
}

func ToViews(customers []sqlc.Customer) []View {
	out := make([]View, 0, len(customers))
	for _, c := range customers {
		out = append(out, ToView(c))
	}
	return out
}
