package seed

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/auth"
	"github.com/benditandayusaputra/depotin/apps/api/internal/customer"
	"github.com/benditandayusaputra/depotin/apps/api/internal/order"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/crypto"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/db"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/idgen"
	"github.com/benditandayusaputra/depotin/apps/api/internal/product"
	"github.com/benditandayusaputra/depotin/apps/api/internal/reminder"
)

const (
	DepotName     = "Depot Tirta Sejuk"
	DepotSlug     = "depot-tirta-sejuk"
	DepotPhone    = "6281200000000"
	OwnerPhone    = "6281200000001"
	CourierPhone  = "6281200000002"
	Courier2Phone = "6281200000003"
	Password      = "demo-depotin-2026"

	randomSeed  = 42
	historyDays = 56
	depotLat    = -6.2615
	depotLng    = 106.7810
)

type pattern int

const (
	patternRegular pattern = iota
	patternIrregular
	patternNew
	patternLapsed
	patternPublic
)

type profile struct {
	name     string
	address  string
	area     string
	usualQty int32
	pattern  pattern
	period   float64
	keepsOne bool
	located  bool
}

type Seeder struct {
	pool  *pgxpool.Pool
	q     *sqlc.Queries
	clock *clock.Fake
	rng   *rand.Rand
	now   time.Time
	loc   *time.Location

	orders    *order.Service
	reminders *reminder.Service
	depot     sqlc.Depot
	owner     sqlc.User
	couriers  []sqlc.User
}

func Run(ctx context.Context, pool *pgxpool.Pool, now time.Time, webOrigin string, linkKey []byte) error {
	q := sqlc.New(pool)
	if existing, err := q.GetDepotBySlug(ctx, DepotSlug); err == nil {
		if err := deleteDepot(ctx, pool, existing.ID); err != nil {
			return err
		}
	} else if !db.IsNoRows(err) {
		return fmt.Errorf("cek depot demo: %w", err)
	}
	loc := time.FixedZone("WIB", 7*3600)
	clk := clock.NewFake(now)
	sealer, err := crypto.NewSealer(linkKey)
	if err != nil {
		return err
	}
	links := customer.NewLinks(webOrigin, sealer)
	s := &Seeder{
		pool: pool, q: q, clock: clk, rng: rand.New(rand.NewSource(randomSeed)), now: now, loc: loc,
		orders: order.NewService(pool, clk, nil), reminders: reminder.NewService(pool, clk, links, nil),
	}
	return s.run(ctx)
}

func Reset(ctx context.Context, pool *pgxpool.Pool, now time.Time, webOrigin string, linkKey []byte) error {
	return Run(ctx, pool, now, webOrigin, linkKey)
}

func deleteDepot(ctx context.Context, pool *pgxpool.Pool, depotID uuid.UUID) error {
	return db.WithTx(ctx, pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		steps := []struct {
			name string
			fn   func(context.Context, uuid.UUID) error
		}{
			{"audit", q.DeleteDepotAuditLogs},
			{"ledger", q.DeleteDepotLedger},
			{"events", q.DeleteDepotOrderEvents},
			{"items", q.DeleteDepotOrderItems},
			{"reminder orders", q.ClearDepotReminderOrders},
			{"orders", q.DeleteDepotOrders},
			{"reminders", q.DeleteDepotReminders},
			{"counters", q.DeleteDepotOrderCounters},
			{"customers", q.DeleteDepotCustomers},
			{"products", q.DeleteDepotProducts},
			{"sessions", q.DeleteDepotSessions},
			{"users", q.DeleteDepotUsers},
			{"depot", q.DeleteDepot},
		}
		for _, step := range steps {
			if err := step.fn(ctx, depotID); err != nil {
				return fmt.Errorf("hapus %s depot demo: %w", step.name, err)
			}
		}
		return nil
	})
}

func (s *Seeder) at(daysAgo float64, hour int) time.Time {
	day := s.now.In(s.loc).AddDate(0, 0, 0)
	base := time.Date(day.Year(), day.Month(), day.Day(), hour, 0, 0, 0, s.loc)
	return base.Add(-time.Duration(daysAgo * 24 * float64(time.Hour)))
}

func (s *Seeder) run(ctx context.Context) error {
	if err := s.createDepot(ctx); err != nil {
		return err
	}
	profiles := s.profiles()
	customers := make([]sqlc.Customer, 0, len(profiles))
	for _, p := range profiles {
		c, err := s.createCustomer(ctx, p)
		if err != nil {
			return err
		}
		customers = append(customers, c)
	}
	for i, p := range profiles {
		if err := s.history(ctx, customers[i], p); err != nil {
			return fmt.Errorf("riwayat %s: %w", p.name, err)
		}
	}
	if err := s.today(ctx, customers, profiles); err != nil {
		return err
	}
	s.clock.Set(s.now)
	if _, err := s.reminders.Refresh(ctx, s.depot); err != nil {
		return err
	}
	return nil
}

func (s *Seeder) createDepot(ctx context.Context) error {
	hash, err := crypto.HashPassword(Password)
	if err != nil {
		return err
	}
	return db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		depot, err := q.CreateDepot(ctx, sqlc.CreateDepotParams{ID: idgen.NewID(), Name: DepotName, Slug: DepotSlug, Phone: DepotPhone, Address: "Jl. Kemanggisan Raya No. 12, Jakarta Barat"})
		if err != nil {
			return fmt.Errorf("buat depot demo: %w", err)
		}
		lat, lng, every := depotLat, depotLng, int32(10)
		if err := q.SetDepotDemo(ctx, sqlc.SetDepotDemoParams{ID: depot.ID, Lat: &lat, Lng: &lng, Address: depot.Address, LoyaltyEvery: &every}); err != nil {
			return fmt.Errorf("tandai depot demo: %w", err)
		}
		depot, err = q.GetDepot(ctx, depot.ID)
		if err != nil {
			return fmt.Errorf("muat depot demo: %w", err)
		}
		s.depot = depot
		users := []struct {
			name, phone, role string
		}{
			{"Bu Sari", OwnerPhone, auth.RoleOwner},
			{"Andi", CourierPhone, auth.RoleCourier},
			{"Budi", Courier2Phone, auth.RoleCourier},
		}
		for _, u := range users {
			created, err := q.CreateUser(ctx, sqlc.CreateUserParams{ID: idgen.NewID(), DepotID: depot.ID, Role: u.role, Name: u.name, Phone: u.phone, PasswordHash: hash})
			if err != nil {
				return fmt.Errorf("buat pengguna demo %s: %w", u.name, err)
			}
			if u.role == auth.RoleOwner {
				s.owner = created
			} else {
				s.couriers = append(s.couriers, created)
			}
		}
		products := []struct {
			name, kind string
			price      int64
			sort       int32
		}{
			{product.DefaultRefillName, product.KindRefill, 6000, 1},
			{"Galon baru + isi", product.KindNewGallon, 45000, 2},
			{"Air mineral dus", product.KindOther, 35000, 3},
		}
		for _, p := range products {
			if _, err := q.CreateProduct(ctx, sqlc.CreateProductParams{ID: idgen.NewID(), DepotID: depot.ID, Name: p.name, Kind: p.kind, Price: p.price, SortOrder: p.sort}); err != nil {
				return fmt.Errorf("buat produk demo: %w", err)
			}
		}
		return nil
	})
}

func (s *Seeder) profiles() []profile {
	names := []string{
		"Bu Rina", "Pak Budi Santoso", "Bu Wati", "Pak Hendra", "Bu Sri Mulyani", "Pak Joko", "Bu Dewi Lestari", "Pak Agus",
		"Bu Yuni", "Pak Slamet", "Bu Ani", "Pak Dedi", "Bu Ratna", "Pak Eko", "Bu Fitri", "Pak Gunawan", "Bu Hani", "Pak Iwan",
		"Bu Lina", "Pak Maman", "Bu Nur", "Pak Oki", "Bu Putri", "Pak Rudi", "Bu Siti", "Pak Tono", "Bu Umi", "Pak Yanto",
		"Kos Melati", "Warung Bu Tini", "Kantor RT 03", "Bu Indah", "Pak Komar", "Bu Mega", "Pak Nanang", "Bu Oca",
		"Pak Pur", "Bu Reni", "Pak Sugeng", "Bu Vina",
	}
	streets := []string{"Jl. Kemanggisan Ilir", "Jl. Anggrek Cakra", "Jl. Rawa Belong", "Jl. Palmerah Utara", "Jl. Kebon Jeruk", "Jl. Sakti IV"}
	areas := []string{"RT 01", "RT 02", "RT 03", "RT 04", "RT 05", "RT 06"}
	periods := []float64{3, 3.5, 4, 5, 6, 7, 10}
	out := make([]profile, 0, len(names))
	for i, name := range names {
		p := profile{
			name:     name,
			address:  fmt.Sprintf("%s No. %d", streets[i%len(streets)], 3+i*2),
			area:     areas[i%len(areas)],
			usualQty: int32(1 + s.rng.Intn(3)),
			period:   periods[s.rng.Intn(len(periods))],
			keepsOne: s.rng.Intn(4) == 0,
			located:  s.rng.Intn(3) != 0,
		}
		switch {
		case i < 18:
			p.pattern = patternRegular
		case i < 26:
			p.pattern = patternIrregular
		case i < 32:
			p.pattern = patternNew
		case i < 36:
			p.pattern = patternLapsed
		default:
			p.pattern = patternPublic
		}
		if name == "Kos Melati" || name == "Kantor RT 03" {
			p.usualQty = 3
		}
		out = append(out, p)
	}
	return out
}

func (s *Seeder) createCustomer(ctx context.Context, p profile) (sqlc.Customer, error) {
	phone := fmt.Sprintf("62812%08d", 30000000+s.rng.Intn(9000000))
	params := sqlc.CreateCustomerParams{
		ID: idgen.NewID(), DepotID: s.depot.ID, Name: p.name, Phone: phone, Address: p.address, Area: p.area,
		AddressNote: s.pick("Pagar hijau", "Sebelah warung", "Rumah pojok", "Lantai 2", ""), Source: customer.SourceOwner, IsVerified: true, UsualQty: p.usualQty,
	}
	if p.pattern == patternPublic {
		params.Source = customer.SourcePublic
		params.IsVerified = false
	}
	if p.located {
		lat := depotLat + (s.rng.Float64()-0.5)*0.02
		lng := depotLng + (s.rng.Float64()-0.5)*0.02
		params.Lat, params.Lng = &lat, &lng
	}
	c, err := s.q.CreateCustomer(ctx, params)
	if err != nil {
		return sqlc.Customer{}, fmt.Errorf("buat pelanggan demo: %w", err)
	}
	return c, nil
}

func (s *Seeder) pick(options ...string) string {
	return options[s.rng.Intn(len(options))]
}

func (s *Seeder) history(ctx context.Context, c sqlc.Customer, p profile) error {
	var moments []float64
	switch p.pattern {
	case patternRegular:
		gap := p.period * float64(p.usualQty)
		t := float64(historyDays) - s.rng.Float64()*gap
		for t > s.rng.Float64()*gap*0.5 {
			moments = append(moments, t)
			t -= gap * (0.85 + s.rng.Float64()*0.3)
		}
	case patternIrregular:
		t := float64(historyDays) - s.rng.Float64()*5
		for t > 1 {
			moments = append(moments, t)
			t -= 2 + s.rng.Float64()*12
		}
	case patternNew:
		moments = append(moments, 1+s.rng.Float64()*9)
	case patternLapsed:
		gap := p.period * float64(p.usualQty)
		t := float64(historyDays)
		stop := 35 + s.rng.Float64()*15
		for t > stop {
			moments = append(moments, t)
			t -= gap
		}
	case patternPublic:
		return nil
	}
	courierIdx := s.rng.Intn(len(s.couriers))
	for i := len(moments) - 1; i >= 0; i-- {
		daysAgo := moments[i]
		returned := p.usualQty
		if p.keepsOne && i == len(moments)-1 {
			returned = p.usualQty - 1
		}
		if p.pattern == patternLapsed && i == 0 {
			returned = 0
		}
		if err := s.deliveredOrder(ctx, c, p.usualQty, daysAgo, s.couriers[courierIdx], returned); err != nil {
			return err
		}
	}
	return nil
}

func (s *Seeder) deliveredOrder(ctx context.Context, c sqlc.Customer, qty int32, daysAgo float64, courier sqlc.User, returned int32) error {
	createdAt := s.at(daysAgo, 8+s.rng.Intn(4))
	s.clock.Set(createdAt)
	source := order.SourceOwner
	actor := order.Actor{Type: order.ActorUser, UserID: &s.owner.ID, Role: auth.RoleOwner}
	if s.rng.Intn(3) == 0 {
		source = order.SourceLink
		actor = order.Actor{Type: order.ActorCustomer}
	}
	var reminderID *uuid.UUID
	if s.rng.Intn(4) == 0 {
		id, err := s.sentReminder(ctx, c, createdAt.Add(-3*time.Hour))
		if err != nil {
			return err
		}
		if id != uuid.Nil {
			reminderID = &id
		}
	}
	detail, _, err := s.orders.Create(ctx, order.CreateInput{
		DepotID: s.depot.ID, CustomerID: c.ID, RefillQty: qty, Fulfilment: order.FulfilmentDelivery,
		Source: source, Status: order.StatusConfirmed, Actor: actor, ReminderID: reminderID,
	})
	if err != nil {
		return err
	}
	if _, err := s.orders.Assign(ctx, s.depot.ID, detail.Order.ID, courier.ID, actor); err != nil {
		return err
	}
	courierActor := order.Actor{Type: order.ActorUser, UserID: &courier.ID, Role: auth.RoleCourier}
	s.clock.Set(createdAt.Add(time.Duration(20+s.rng.Intn(40)) * time.Minute))
	if _, err := s.orders.Dispatch(ctx, s.depot.ID, detail.Order.ID, courierActor, ""); err != nil {
		return err
	}
	s.clock.Set(s.clock.Now().Add(time.Duration(15+s.rng.Intn(60)) * time.Minute))
	method := s.pick(order.PaymentCash, order.PaymentCash, order.PaymentTransfer, order.PaymentQRIS)
	if _, err := s.orders.Deliver(ctx, s.depot.ID, detail.Order.ID, order.DeliverInput{GallonsReturned: returned, PaymentMethod: method, Paid: true}, courierActor, ""); err != nil {
		return err
	}
	return nil
}

func (s *Seeder) sentReminder(ctx context.Context, c sqlc.Customer, sentAt time.Time) (uuid.UUID, error) {
	day := time.Date(sentAt.In(s.loc).Year(), sentAt.In(s.loc).Month(), sentAt.In(s.loc).Day(), 0, 0, 0, 0, time.UTC)
	r, err := s.q.InsertReminder(ctx, sqlc.InsertReminderParams{
		ID: idgen.NewID(), DepotID: s.depot.ID, CustomerID: c.ID, DueDate: day, PredictedEmptyAt: sentAt.Add(24 * time.Hour),
		Status: reminder.StatusSent, SentAt: &sentAt, SentBy: &s.owner.ID,
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return uuid.Nil, nil
		}
		return uuid.Nil, fmt.Errorf("sisipkan pengingat demo: %w", err)
	}
	if s.rng.Intn(3) == 0 {
		if _, err := s.q.InsertReminder(ctx, sqlc.InsertReminderParams{
			ID: idgen.NewID(), DepotID: s.depot.ID, CustomerID: c.ID, DueDate: day.AddDate(0, 0, -4), PredictedEmptyAt: sentAt.Add(-72 * time.Hour),
			Status: reminder.StatusSent, SentAt: ptrTime(sentAt.Add(-96 * time.Hour)), SentBy: &s.owner.ID,
		}); err != nil && !db.IsUniqueViolation(err) {
			return uuid.Nil, fmt.Errorf("sisipkan pengingat lewat demo: %w", err)
		}
	}
	return r.ID, nil
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func (s *Seeder) today(ctx context.Context, customers []sqlc.Customer, profiles []profile) error {
	ownerActor := order.Actor{Type: order.ActorUser, UserID: &s.owner.ID, Role: auth.RoleOwner}
	regular := make([]sqlc.Customer, 0)
	public := make([]sqlc.Customer, 0)
	for i, p := range profiles {
		switch p.pattern {
		case patternRegular, patternIrregular:
			regular = append(regular, customers[i])
		case patternPublic:
			public = append(public, customers[i])
		}
	}

	for i := range 3 {
		if err := s.deliveredOrder(ctx, regular[i], profiles[i].usualQty, 0.25+float64(i)*0.05, s.couriers[i%2], profiles[i].usualQty); err != nil {
			return fmt.Errorf("pesanan selesai hari ini: %w", err)
		}
	}

	s.clock.Set(s.at(0, 7))
	for i := 3; i < 6; i++ {
		detail, _, err := s.orders.Create(ctx, order.CreateInput{
			DepotID: s.depot.ID, CustomerID: regular[i].ID, Fulfilment: order.FulfilmentDelivery, Source: order.SourceOwner, Status: order.StatusConfirmed, Actor: ownerActor,
		})
		if err != nil {
			return fmt.Errorf("pesanan dikonfirmasi hari ini: %w", err)
		}
		if _, err := s.orders.Assign(ctx, s.depot.ID, detail.Order.ID, s.couriers[0].ID, ownerActor); err != nil {
			return err
		}
		if i >= 5 {
			s.clock.Set(s.at(0, 8))
			courierActor := order.Actor{Type: order.ActorUser, UserID: &s.couriers[0].ID, Role: auth.RoleCourier}
			if _, err := s.orders.Dispatch(ctx, s.depot.ID, detail.Order.ID, courierActor, ""); err != nil {
				return err
			}
		}
	}

	s.clock.Set(s.at(0, 9))
	detail, _, err := s.orders.Create(ctx, order.CreateInput{
		DepotID: s.depot.ID, CustomerID: regular[6].ID, Fulfilment: order.FulfilmentDelivery, Source: order.SourceOwner, Status: order.StatusConfirmed, Actor: ownerActor,
	})
	if err != nil {
		return err
	}
	if _, err := s.orders.Assign(ctx, s.depot.ID, detail.Order.ID, s.couriers[1].ID, ownerActor); err != nil {
		return err
	}
	cancelled, _, err := s.orders.Create(ctx, order.CreateInput{
		DepotID: s.depot.ID, CustomerID: regular[7].ID, Fulfilment: order.FulfilmentDelivery, Source: order.SourceLink, Status: order.StatusConfirmed, Actor: order.Actor{Type: order.ActorCustomer},
	})
	if err != nil {
		return err
	}
	if _, err := s.orders.Cancel(ctx, s.depot.ID, cancelled.Order.ID, "Pelanggan sedang di luar kota", ownerActor); err != nil {
		return err
	}

	for i, c := range public[:2] {
		s.clock.Set(s.at(0, 9).Add(time.Duration(i*17) * time.Minute))
		if _, _, err := s.orders.Create(ctx, order.CreateInput{
			DepotID: s.depot.ID, CustomerID: c.ID, RefillQty: 1, Fulfilment: order.FulfilmentDelivery, Source: order.SourcePublic, Status: order.StatusPending,
			Actor: order.Actor{Type: order.ActorCustomer}, Delivery: &order.DeliveryCopy{Name: c.Name, Phone: c.Phone, Address: c.Address, Note: c.AddressNote},
		}); err != nil {
			return fmt.Errorf("pesanan menunggu hari ini: %w", err)
		}
	}
	return nil
}
