package report

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/audit"
	"github.com/benditandayusaputra/depotin/apps/api/internal/order"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
	"github.com/benditandayusaputra/depotin/apps/api/internal/reminder"
)

const (
	maxRangeDays     = 92
	defaultRangeDays = 30
)

type Today struct {
	Date            string           `json:"date"`
	OrdersByStatus  map[string]int64 `json:"orders_by_status"`
	Revenue         int64            `json:"revenue"`
	GallonsSold     int64            `json:"gallons_sold"`
	DeliveredOrders int64            `json:"delivered_orders"`
	ExpectedDemand  int64            `json:"expected_demand"`
	RemindersQueued int64            `json:"reminders_queued"`
	GallonsOnLoan   int64            `json:"gallons_on_loan"`
	CustomersDue    int64            `json:"customers_due"`
}

type Summary struct {
	From               string           `json:"from"`
	To                 string           `json:"to"`
	Revenue            int64            `json:"revenue"`
	GallonsSold        int64            `json:"gallons_sold"`
	DeliveredOrders    int64            `json:"delivered_orders"`
	OrdersBySource     map[string]int64 `json:"orders_by_source"`
	RemindersSent      int64            `json:"reminders_sent"`
	RemindersConverted int64            `json:"reminders_converted"`
	ConversionRate     float64          `json:"conversion_rate"`
	NewCustomers       int64            `json:"new_customers"`
	ActiveCustomers    int64            `json:"active_customers"`
	Daily              []DailyPoint     `json:"daily"`
}

type DailyPoint struct {
	Day     string `json:"day"`
	Revenue int64  `json:"revenue"`
	Gallons int64  `json:"gallons"`
}

type Handler struct {
	q         *sqlc.Queries
	clock     clock.Clock
	reminders *reminder.Service
}

func NewHandler(pool *pgxpool.Pool, clk clock.Clock, reminders *reminder.Service) *Handler {
	return &Handler{q: sqlc.New(pool), clock: clk, reminders: reminders}
}

func (h *Handler) Register(r fiber.Router, ownerOnly fiber.Handler) {
	r.Get("/dashboard/today", ownerOnly, h.today)
	r.Get("/reports/summary", ownerOnly, h.summary)
	r.Get("/reports/export.csv", ownerOnly, h.exportCSV)
	r.Get("/audit", ownerOnly, h.auditLogs)
}

type window struct {
	from, to time.Time
	fromDay  time.Time
	toDay    time.Time
	loc      *time.Location
}

func (h *Handler) depot(c fiber.Ctx) (sqlc.Depot, error) {
	p, _ := httpx.CurrentPrincipal(c)
	depot, err := h.q.GetDepot(c.Context(), p.DepotID)
	if err != nil {
		return sqlc.Depot{}, fmt.Errorf("ambil depot: %w", err)
	}
	return depot, nil
}

func dayRange(day time.Time, loc *time.Location) (time.Time, time.Time) {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	return start, start.AddDate(0, 0, 1)
}

func (h *Handler) today(c fiber.Ctx) error {
	depot, err := h.depot(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	ctx := c.Context()
	loc := order.DepotLocation(depot)
	today := order.DateIn(h.clock.Now(), loc)
	from, to := dayRange(today, loc)

	byStatus, err := h.q.CountOrdersByStatusForDate(ctx, sqlc.CountOrdersByStatusForDateParams{DepotID: depot.ID, Day: today})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("pesanan per status: %w", err))
	}
	totals, err := h.q.DeliveredTotals(ctx, sqlc.DeliveredTotalsParams{DepotID: depot.ID, FromAt: from, ToAt: to})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("total hari ini: %w", err))
	}
	demand, err := h.q.ExpectedDemandToday(ctx, sqlc.ExpectedDemandTodayParams{DepotID: depot.ID, Before: to})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("perkiraan permintaan: %w", err))
	}
	queued, err := h.reminders.QueuedCount(ctx, depot)
	if err != nil {
		return httpx.Fail(c, err)
	}
	onLoan, err := h.q.SumLoanBalance(ctx, depot.ID)
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("galon dipinjam: %w", err))
	}
	due, err := h.q.ListDueCustomers(ctx, sqlc.ListDueCustomersParams{DepotID: depot.ID, Limit: httpx.MaxLimit, DueBefore: to.AddDate(0, 0, int(depot.ReminderLeadDays))})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("pelanggan jatuh tempo: %w", err))
	}

	out := Today{
		Date: today.Format(time.DateOnly), OrdersByStatus: map[string]int64{},
		Revenue: totals.Revenue, GallonsSold: totals.Gallons, DeliveredOrders: totals.Orders,
		ExpectedDemand: demand, RemindersQueued: queued, GallonsOnLoan: onLoan, CustomersDue: int64(len(due)),
	}
	for _, s := range []string{order.StatusPending, order.StatusConfirmed, order.StatusOnDelivery, order.StatusDelivered, order.StatusCancelled} {
		out.OrdersByStatus[s] = 0
	}
	for _, row := range byStatus {
		out.OrdersByStatus[row.Status] = row.Total
	}
	return httpx.OK(c, out)
}

func (h *Handler) parseRange(c fiber.Ctx, depot sqlc.Depot) (window, error) {
	loc := order.DepotLocation(depot)
	today := order.DateIn(h.clock.Now(), loc)
	toDay := today
	fromDay := today.AddDate(0, 0, -defaultRangeDays+1)
	if raw := c.Query("from"); raw != "" {
		d, err := time.Parse(time.DateOnly, raw)
		if err != nil {
			return window{}, httpx.Validation(map[string]string{"from": "Format tanggal harus YYYY-MM-DD."})
		}
		fromDay = d
	}
	if raw := c.Query("to"); raw != "" {
		d, err := time.Parse(time.DateOnly, raw)
		if err != nil {
			return window{}, httpx.Validation(map[string]string{"to": "Format tanggal harus YYYY-MM-DD."})
		}
		toDay = d
	}
	if toDay.Before(fromDay) {
		return window{}, httpx.Validation(map[string]string{"to": "Tanggal akhir harus setelah tanggal awal."})
	}
	if toDay.Sub(fromDay) > maxRangeDays*24*time.Hour {
		return window{}, httpx.Validation(map[string]string{"to": "Rentang maksimal 92 hari."})
	}
	from, _ := dayRange(fromDay, loc)
	_, to := dayRange(toDay, loc)
	return window{from: from, to: to, fromDay: fromDay, toDay: toDay, loc: loc}, nil
}

func (h *Handler) summary(c fiber.Ctx) error {
	depot, err := h.depot(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	w, err := h.parseRange(c, depot)
	if err != nil {
		return httpx.Fail(c, err)
	}
	ctx := c.Context()
	totals, err := h.q.DeliveredTotals(ctx, sqlc.DeliveredTotalsParams{DepotID: depot.ID, FromAt: w.from, ToAt: w.to})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("total rentang: %w", err))
	}
	bySource, err := h.q.CountOrdersBySource(ctx, sqlc.CountOrdersBySourceParams{DepotID: depot.ID, FromAt: w.from, ToAt: w.to})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("pesanan per sumber: %w", err))
	}
	stats, err := h.q.ReminderStats(ctx, sqlc.ReminderStatsParams{DepotID: depot.ID, FromAt: w.from, ToAt: w.to})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("statistik pengingat: %w", err))
	}
	newCustomers, err := h.q.CountNewCustomers(ctx, sqlc.CountNewCustomersParams{DepotID: depot.ID, FromAt: w.from, ToAt: w.to})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("pelanggan baru: %w", err))
	}
	daily, err := h.q.ListDailyRevenue(ctx, sqlc.ListDailyRevenueParams{DepotID: depot.ID, FromAt: w.from, ToAt: w.to, Tz: depot.Timezone})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("omzet harian: %w", err))
	}

	out := Summary{
		From: w.fromDay.Format(time.DateOnly), To: w.toDay.Format(time.DateOnly),
		Revenue: totals.Revenue, GallonsSold: totals.Gallons, DeliveredOrders: totals.Orders,
		OrdersBySource: map[string]int64{}, RemindersSent: stats.Sent, RemindersConverted: stats.Ordered,
		NewCustomers: newCustomers, ActiveCustomers: totals.ActiveCustomers, Daily: make([]DailyPoint, 0, len(daily)),
	}
	for _, s := range []string{order.SourcePublic, order.SourceLink, order.SourceReminder, order.SourceOwner, order.SourceCourier} {
		out.OrdersBySource[s] = 0
	}
	for _, row := range bySource {
		out.OrdersBySource[row.Source] = row.Total
	}
	if stats.Sent > 0 {
		out.ConversionRate = float64(stats.Ordered) / float64(stats.Sent)
	}
	for _, d := range daily {
		out.Daily = append(out.Daily, DailyPoint{Day: d.Day.Format(time.DateOnly), Revenue: d.Revenue, Gallons: d.Gallons})
	}
	return httpx.OK(c, out)
}

func (h *Handler) exportCSV(c fiber.Ctx) error {
	depot, err := h.depot(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	w, err := h.parseRange(c, depot)
	if err != nil {
		return httpx.Fail(c, err)
	}
	rows, err := h.q.ListDeliveredOrdersForExport(c.Context(), sqlc.ListDeliveredOrdersForExportParams{DepotID: depot.ID, FromAt: w.from, ToAt: w.to})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("data ekspor: %w", err))
	}
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	_ = writer.Write([]string{"kode", "tanggal", "nama", "nomor", "galon", "gratis", "total", "cara_bayar", "status_bayar", "sumber", "kurir"})
	for _, r := range rows {
		method, courier := "", ""
		if r.PaymentMethod != nil {
			method = *r.PaymentMethod
		}
		if r.CourierName != nil {
			courier = *r.CourierName
		}
		deliveredAt := ""
		if r.DeliveredAt != nil {
			deliveredAt = r.DeliveredAt.In(w.loc).Format("2006-01-02 15:04")
		}
		_ = writer.Write([]string{
			r.Code, deliveredAt, r.DeliveryName, r.DeliveryPhone, strconv.Itoa(int(r.RefillQty)), strconv.Itoa(int(r.FreeQty)),
			strconv.FormatInt(r.Total, 10), method, r.PaymentStatus, r.Source, courier,
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return httpx.Fail(c, fmt.Errorf("tulis csv: %w", err))
	}
	p, _ := httpx.CurrentPrincipal(c)
	if err := audit.Record(c.Context(), h.q, audit.Entry{
		DepotID: depot.ID, UserID: &p.UserID, Action: audit.ActionReportExported, EntityType: "report",
		Meta: map[string]any{"from": w.fromDay.Format(time.DateOnly), "to": w.toDay.Format(time.DateOnly), "rows": len(rows)}, IP: httpx.ClientIP(c),
	}); err != nil {
		return httpx.Fail(c, err)
	}
	filename := fmt.Sprintf("depotin-%s-%s-%s.csv", depot.Slug, w.fromDay.Format("20060102"), w.toDay.Format("20060102"))
	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+filename+`"`)
	c.Set(fiber.HeaderCacheControl, "no-store")
	if err := c.Send(buf.Bytes()); err != nil {
		return fmt.Errorf("kirim csv: %w", err)
	}
	return nil
}

type AuditView struct {
	ID         string         `json:"id"`
	UserID     *string        `json:"user_id"`
	Action     string         `json:"action"`
	EntityType string         `json:"entity_type"`
	EntityID   *string        `json:"entity_id"`
	Meta       map[string]any `json:"meta"`
	CreatedAt  time.Time      `json:"created_at"`
}

func (h *Handler) auditLogs(c fiber.Ctx) error {
	page, err := httpx.ParsePage(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	p, _ := httpx.CurrentPrincipal(c)
	params := sqlc.ListAuditLogsParams{DepotID: p.DepotID, Limit: int32(page.Limit) + 1}
	if page.HasCursor {
		params.BeforeAt = &page.Cursor.At
		params.BeforeID = &page.Cursor.ID
	}
	rows, err := h.q.ListAuditLogs(c.Context(), params)
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("log audit: %w", err))
	}
	next := ""
	if len(rows) > page.Limit {
		rows = rows[:page.Limit]
		last := rows[len(rows)-1]
		next = httpx.EncodeCursor(last.CreatedAt, last.ID)
	}
	views := make([]AuditView, 0, len(rows))
	for _, r := range rows {
		v := AuditView{ID: r.ID.String(), Action: r.Action, EntityType: r.EntityType, Meta: map[string]any{}, CreatedAt: r.CreatedAt}
		if r.UserID != nil {
			s := r.UserID.String()
			v.UserID = &s
		}
		if r.EntityID != nil {
			s := r.EntityID.String()
			v.EntityID = &s
		}
		_ = json.Unmarshal(r.Meta, &v.Meta)
		views = append(views, v)
	}
	return httpx.List(c, views, next)
}
