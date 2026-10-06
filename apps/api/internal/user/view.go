package user

import (
	"time"

	"github.com/google/uuid"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
)

type View struct {
	ID        uuid.UUID  `json:"id"`
	DepotID   uuid.UUID  `json:"depot_id"`
	Role      string     `json:"role"`
	Name      string     `json:"name"`
	Phone     string     `json:"phone"`
	IsActive  bool       `json:"is_active"`
	LastLogin *time.Time `json:"last_login_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func ToView(u sqlc.User) View {
	return View{
		ID: u.ID, DepotID: u.DepotID, Role: u.Role, Name: u.Name, Phone: u.Phone,
		IsActive: u.IsActive, LastLogin: u.LastLoginAt, CreatedAt: u.CreatedAt,
	}
}

func ToViews(users []sqlc.User) []View {
	out := make([]View, 0, len(users))
	for _, u := range users {
		out = append(out, ToView(u))
	}
	return out
}
