package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/google/uuid"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/idgen"
)

const (
	ActionLoginSuccess      = "login_success"
	ActionLoginFailed       = "login_failed"
	ActionPasswordChanged   = "password_changed"
	ActionUserCreated       = "user_created"
	ActionUserUpdated       = "user_updated"
	ActionUserPasswordReset = "user_password_reset"
	ActionPriceChanged      = "price_changed"
	ActionLedgerAdjusted    = "ledger_adjusted"
	ActionLinkRotated       = "link_rotated"
	ActionOrderCancelled    = "order_cancelled"
	ActionReportExported    = "report_exported"
)

type Entry struct {
	DepotID    uuid.UUID
	UserID     *uuid.UUID
	Action     string
	EntityType string
	EntityID   *uuid.UUID
	Meta       map[string]any
	IP         string
}

func Record(ctx context.Context, q *sqlc.Queries, e Entry) error {
	meta := []byte("{}")
	if len(e.Meta) > 0 {
		encoded, err := json.Marshal(e.Meta)
		if err != nil {
			return fmt.Errorf("encode meta audit: %w", err)
		}
		meta = encoded
	}
	var ip *string
	if addr, err := netip.ParseAddr(e.IP); err == nil {
		s := addr.String()
		ip = &s
	}
	if err := q.InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		ID:         idgen.NewID(),
		DepotID:    e.DepotID,
		UserID:     e.UserID,
		Action:     e.Action,
		EntityType: e.EntityType,
		EntityID:   e.EntityID,
		Meta:       meta,
		Ip:         ip,
	}); err != nil {
		return fmt.Errorf("simpan audit: %w", err)
	}
	return nil
}
