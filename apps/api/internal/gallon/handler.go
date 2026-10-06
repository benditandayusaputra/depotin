package gallon

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/audit"
	"github.com/benditandayusaputra/depotin/apps/api/internal/customer"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/db"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/idgen"
)

type EntryView struct {
	ID           uuid.UUID  `json:"id"`
	OrderID      *uuid.UUID `json:"order_id"`
	Kind         string     `json:"kind"`
	Delta        int32      `json:"delta"`
	BalanceAfter int32      `json:"balance_after"`
	Note         string     `json:"note"`
	CreatedAt    time.Time  `json:"created_at"`
}

func toEntryView(e sqlc.GallonLedger) EntryView {
	return EntryView{ID: e.ID, OrderID: e.OrderID, Kind: e.Kind, Delta: e.Delta, BalanceAfter: e.BalanceAfter, Note: e.Note, CreatedAt: e.CreatedAt}
}

type Summary struct {
	TotalOnLoan int64           `json:"total_on_loan"`
	Customers   int64           `json:"customers_with_loan"`
	Idle        []customer.View `json:"idle"`
}

type Handler struct {
	pool  *pgxpool.Pool
	q     *sqlc.Queries
	clock clock.Clock
}

func NewHandler(pool *pgxpool.Pool, clk clock.Clock) *Handler {
	return &Handler{pool: pool, q: sqlc.New(pool), clock: clk}
}

func (h *Handler) Register(r fiber.Router, ownerOnly fiber.Handler) {
	r.Get("/customers/:id/ledger", ownerOnly, h.ledger)
	r.Post("/customers/:id/ledger-adjustments", ownerOnly, h.adjust)
	r.Get("/gallons/summary", ownerOnly, h.summary)
}

func (h *Handler) ledger(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.Fail(c, httpx.NotFound(""))
	}
	page, err := httpx.ParsePage(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	p, _ := httpx.CurrentPrincipal(c)
	if _, err := h.q.GetCustomer(c.Context(), sqlc.GetCustomerParams{ID: id, DepotID: p.DepotID}); err != nil {
		if db.IsNoRows(err) {
			return httpx.Fail(c, httpx.NotFound(""))
		}
		return httpx.Fail(c, fmt.Errorf("cari pelanggan: %w", err))
	}
	params := sqlc.ListLedgerByCustomerParams{CustomerID: id, DepotID: p.DepotID, Limit: int32(page.Limit) + 1}
	if page.HasCursor {
		params.BeforeAt = &page.Cursor.At
		params.BeforeID = &page.Cursor.ID
	}
	rows, err := h.q.ListLedgerByCustomer(c.Context(), params)
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("buku galon: %w", err))
	}
	next := ""
	if len(rows) > page.Limit {
		rows = rows[:page.Limit]
		last := rows[len(rows)-1]
		next = httpx.EncodeCursor(last.CreatedAt, last.ID)
	}
	views := make([]EntryView, 0, len(rows))
	for _, r := range rows {
		views = append(views, toEntryView(r))
	}
	return httpx.List(c, views, next)
}

type adjustRequest struct {
	Kind  string `json:"kind" validate:"required,oneof=adjustment lost"`
	Delta int32  `json:"delta" validate:"required,gte=-100,lte=100"`
	Note  string `json:"note" validate:"required,min=3,max=200"`
}

func (h *Handler) adjust(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.Fail(c, httpx.NotFound(""))
	}
	var req adjustRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	p, _ := httpx.CurrentPrincipal(c)
	var entry sqlc.GallonLedger
	err = db.WithTx(c.Context(), h.pool, func(tx pgx.Tx) error {
		q := h.q.WithTx(tx)
		cust, err := q.GetCustomerForUpdate(c.Context(), sqlc.GetCustomerForUpdateParams{ID: id, DepotID: p.DepotID})
		if err != nil {
			if db.IsNoRows(err) {
				return httpx.NotFound("")
			}
			return fmt.Errorf("kunci pelanggan: %w", err)
		}
		next, err := Apply(cust.LoanBalance, req.Delta)
		if err != nil {
			return err
		}
		entry, err = q.InsertLedger(c.Context(), sqlc.InsertLedgerParams{
			ID: idgen.NewID(), DepotID: p.DepotID, CustomerID: cust.ID, Kind: req.Kind, Delta: req.Delta, BalanceAfter: next, Note: req.Note, CreatedBy: &p.UserID,
		})
		if err != nil {
			return fmt.Errorf("tulis buku galon: %w", err)
		}
		if err := q.SetCustomerLoanBalance(c.Context(), sqlc.SetCustomerLoanBalanceParams{ID: cust.ID, DepotID: p.DepotID, LoanBalance: next}); err != nil {
			return fmt.Errorf("perbarui saldo galon: %w", err)
		}
		return audit.Record(c.Context(), q, audit.Entry{
			DepotID: p.DepotID, UserID: &p.UserID, Action: audit.ActionLedgerAdjusted, EntityType: "customer", EntityID: &cust.ID,
			Meta: map[string]any{"kind": req.Kind, "delta": req.Delta, "balance_after": next}, IP: httpx.ClientIP(c),
		})
	})
	if err != nil {
		if errors.Is(err, ErrNegativeBalance) {
			return httpx.Fail(c, httpx.Validation(map[string]string{"delta": "Saldo galon tidak boleh negatif."}))
		}
		return httpx.Fail(c, err)
	}
	return httpx.Created(c, toEntryView(entry))
}

func (h *Handler) summary(c fiber.Ctx) error {
	p, _ := httpx.CurrentPrincipal(c)
	total, err := h.q.SumLoanBalance(c.Context(), p.DepotID)
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("total galon dipinjam: %w", err))
	}
	withLoan, err := h.q.ListLoanCustomers(c.Context(), sqlc.ListLoanCustomersParams{DepotID: p.DepotID, Limit: httpx.MaxLimit})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("pelanggan memegang galon: %w", err))
	}
	idleBefore := h.clock.Now().AddDate(0, 0, -IdleAfterDays)
	idle, err := h.q.ListIdleLoanCustomers(c.Context(), sqlc.ListIdleLoanCustomersParams{DepotID: p.DepotID, Limit: httpx.MaxLimit, IdleBefore: idleBefore})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("galon mengendap: %w", err))
	}
	return httpx.OK(c, Summary{TotalOnLoan: total, Customers: int64(len(withLoan)), Idle: customer.ToViews(idle)})
}
