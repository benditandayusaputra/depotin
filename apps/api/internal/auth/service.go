package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/audit"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/crypto"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/db"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/idgen"
	"github.com/benditandayusaputra/depotin/apps/api/internal/product"
)

const (
	MaxFailedLogins   = 5
	LockDuration      = 15 * time.Minute
	refreshTokenBytes = 32
	maxUserAgentChars = 200
	maxSlugAttempts   = 20
	maxSlugRetries    = 3
)

var (
	ErrInvalidCredentials = errors.New("nomor atau kata sandi salah")
	ErrAccountLocked      = errors.New("akun terkunci")
	ErrPhoneTaken         = errors.New("nomor sudah terdaftar")
	ErrUserInactive       = errors.New("akun tidak aktif")
)

type Tokens struct {
	Access  string
	Refresh string
}

type RequestMeta struct {
	UserAgent string
	IP        string
}

type RegisterInput struct {
	DepotName string
	OwnerName string
	Phone     string
	Password  string
	Meta      RequestMeta
}

type Service struct {
	pool   *pgxpool.Pool
	q      *sqlc.Queries
	clock  clock.Clock
	tokens *TokenIssuer
}

func NewService(pool *pgxpool.Pool, clk clock.Clock, tokens *TokenIssuer) *Service {
	return &Service{pool: pool, q: sqlc.New(pool), clock: clk, tokens: tokens}
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (sqlc.User, sqlc.Depot, Tokens, error) {
	hash, err := crypto.HashPassword(in.Password)
	if err != nil {
		return sqlc.User{}, sqlc.Depot{}, Tokens{}, err
	}
	var user sqlc.User
	var depot sqlc.Depot
	var tokens Tokens
	for attempt := 0; ; attempt++ {
		err = s.registerTx(ctx, in, hash, attempt > 0, &user, &depot, &tokens)
		if err == nil || attempt >= maxSlugRetries || !isSlugCollision(err) {
			break
		}
	}
	if err != nil {
		return sqlc.User{}, sqlc.Depot{}, Tokens{}, err
	}
	return user, depot, tokens, nil
}

func isSlugCollision(err error) bool {
	return db.IsUniqueViolation(err) && db.ConstraintName(err) == "depots_slug_key"
}

func (s *Service) registerTx(ctx context.Context, in RegisterInput, hash string, forceSuffix bool, user *sqlc.User, depot *sqlc.Depot, tokens *Tokens) error {
	return db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		slug, err := s.uniqueSlug(ctx, q, in.DepotName, forceSuffix)
		if err != nil {
			return err
		}
		*depot, err = q.CreateDepot(ctx, sqlc.CreateDepotParams{
			ID: idgen.NewID(), Name: in.DepotName, Slug: slug, Phone: in.Phone,
		})
		if err != nil {
			return fmt.Errorf("buat depot: %w", err)
		}
		*user, err = q.CreateUser(ctx, sqlc.CreateUserParams{
			ID: idgen.NewID(), DepotID: depot.ID, Role: RoleOwner, Name: in.OwnerName, Phone: in.Phone, PasswordHash: hash,
		})
		if err != nil {
			if db.IsUniqueViolation(err) {
				return ErrPhoneTaken
			}
			return fmt.Errorf("buat pemilik: %w", err)
		}
		if _, err := q.CreateProduct(ctx, sqlc.CreateProductParams{
			ID: idgen.NewID(), DepotID: depot.ID, Name: product.DefaultRefillName, Kind: product.KindRefill, Price: product.DefaultRefillPrice,
		}); err != nil {
			return fmt.Errorf("buat produk bawaan: %w", err)
		}
		*tokens, err = s.startSession(ctx, q, *user, idgen.NewID(), in.Meta)
		return err
	})
}

func (s *Service) uniqueSlug(ctx context.Context, q *sqlc.Queries, name string, forceSuffix bool) (string, error) {
	base := Slugify(name)
	if forceSuffix {
		suffix, err := idgen.RandomToken(4)
		if err != nil {
			return "", err
		}
		return base + "-" + strings.ToLower(suffix), nil
	}
	candidate := base
	for i := 2; i <= maxSlugAttempts; i++ {
		exists, err := q.SlugExists(ctx, candidate)
		if err != nil {
			return "", fmt.Errorf("cek slug: %w", err)
		}
		if !exists {
			return candidate, nil
		}
		candidate = base + "-" + strconv.Itoa(i)
	}
	suffix, err := idgen.RandomToken(4)
	if err != nil {
		return "", err
	}
	return base + "-" + strings.ToLower(suffix), nil
}

func (s *Service) Login(ctx context.Context, phoneNumber, password string, meta RequestMeta) (sqlc.User, Tokens, error) {
	now := s.clock.Now()
	user, err := s.q.GetUserByPhone(ctx, phoneNumber)
	if err != nil {
		if db.IsNoRows(err) {
			_, _ = crypto.VerifyPassword(crypto.DummyHash(), password)
			return sqlc.User{}, Tokens{}, ErrInvalidCredentials
		}
		return sqlc.User{}, Tokens{}, fmt.Errorf("cari pengguna: %w", err)
	}
	if user.LockedUntil != nil && user.LockedUntil.After(now) {
		_, _ = crypto.VerifyPassword(user.PasswordHash, password)
		return sqlc.User{}, Tokens{}, ErrAccountLocked
	}
	ok, err := crypto.VerifyPassword(user.PasswordHash, password)
	if err != nil {
		return sqlc.User{}, Tokens{}, fmt.Errorf("verifikasi kata sandi: %w", err)
	}
	if !ok {
		return sqlc.User{}, Tokens{}, s.recordFailure(ctx, user, now, meta)
	}
	if !user.IsActive {
		return sqlc.User{}, Tokens{}, ErrUserInactive
	}

	var tokens Tokens
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.RecordLoginSuccess(ctx, sqlc.RecordLoginSuccessParams{ID: user.ID, LastLoginAt: &now}); err != nil {
			return fmt.Errorf("catat login: %w", err)
		}
		if err := audit.Record(ctx, q, audit.Entry{DepotID: user.DepotID, UserID: &user.ID, Action: audit.ActionLoginSuccess, EntityType: "user", EntityID: &user.ID, IP: meta.IP}); err != nil {
			return err
		}
		tokens, err = s.startSession(ctx, q, user, idgen.NewID(), meta)
		return err
	})
	if err != nil {
		return sqlc.User{}, Tokens{}, err
	}
	return user, tokens, nil
}

func (s *Service) recordFailure(ctx context.Context, user sqlc.User, now time.Time, meta RequestMeta) error {
	locked := false
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		result, err := q.RecordLoginFailure(ctx, sqlc.RecordLoginFailureParams{
			ID: user.ID, MaxFailures: MaxFailedLogins, LockUntil: now.Add(LockDuration),
		})
		if err != nil {
			return fmt.Errorf("catat gagal login: %w", err)
		}
		if err := audit.Record(ctx, q, audit.Entry{DepotID: user.DepotID, UserID: &user.ID, Action: audit.ActionLoginFailed, EntityType: "user", EntityID: &user.ID, Meta: map[string]any{"failed_count": result.FailedLoginCount}, IP: meta.IP}); err != nil {
			return err
		}
		locked = result.LockedUntil != nil && result.LockedUntil.After(now)
		return nil
	})
	if err != nil {
		return err
	}
	if locked {
		return ErrAccountLocked
	}
	return ErrInvalidCredentials
}

func (s *Service) startSession(ctx context.Context, q *sqlc.Queries, user sqlc.User, familyID uuid.UUID, meta RequestMeta) (Tokens, error) {
	now := s.clock.Now()
	refresh, err := idgen.RandomToken(refreshTokenBytes)
	if err != nil {
		return Tokens{}, err
	}
	ua := meta.UserAgent
	if len(ua) > maxUserAgentChars {
		ua = ua[:maxUserAgentChars]
	}
	if _, err := q.CreateSession(ctx, sqlc.CreateSessionParams{
		ID: idgen.NewID(), UserID: user.ID, FamilyID: familyID, TokenHash: crypto.HashToken(refresh),
		ExpiresAt: now.Add(RefreshTokenTTL), UserAgent: ua, Ip: db.InetFromIP(meta.IP),
	}); err != nil {
		return Tokens{}, fmt.Errorf("buat sesi: %w", err)
	}
	access, err := s.tokens.Issue(user.ID, user.DepotID, familyID, user.Role, now)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{Access: access, Refresh: refresh}, nil
}

func (s *Service) Refresh(ctx context.Context, rawRefresh string, meta RequestMeta) (Tokens, error) {
	now := s.clock.Now()
	sess, err := s.q.GetSessionByTokenHash(ctx, crypto.HashToken(rawRefresh))
	if err != nil {
		if db.IsNoRows(err) {
			return Tokens{}, ErrInvalidToken
		}
		return Tokens{}, fmt.Errorf("cari sesi: %w", err)
	}
	if sess.RevokedAt != nil || sess.ExpiresAt.Before(now) {
		return Tokens{}, ErrInvalidToken
	}
	if sess.RotatedAt != nil {
		if err := s.q.RevokeSessionFamily(ctx, sqlc.RevokeSessionFamilyParams{FamilyID: sess.FamilyID, RevokedAt: &now}); err != nil {
			return Tokens{}, fmt.Errorf("cabut keluarga sesi: %w", err)
		}
		return Tokens{}, ErrInvalidToken
	}
	user, err := s.q.GetUser(ctx, sess.UserID)
	if err != nil {
		return Tokens{}, fmt.Errorf("cari pengguna sesi: %w", err)
	}
	if !user.IsActive {
		return Tokens{}, ErrInvalidToken
	}
	var tokens Tokens
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		rows, err := q.MarkSessionRotated(ctx, sqlc.MarkSessionRotatedParams{ID: sess.ID, RotatedAt: &now})
		if err != nil {
			return fmt.Errorf("tandai rotasi: %w", err)
		}
		if rows == 0 {
			return ErrInvalidToken
		}
		tokens, err = s.startSession(ctx, q, user, sess.FamilyID, meta)
		return err
	})
	if err != nil {
		return Tokens{}, err
	}
	return tokens, nil
}

func (s *Service) Logout(ctx context.Context, rawRefresh string) error {
	if rawRefresh == "" {
		return nil
	}
	now := s.clock.Now()
	sess, err := s.q.GetSessionByTokenHash(ctx, crypto.HashToken(rawRefresh))
	if err != nil {
		if db.IsNoRows(err) {
			return nil
		}
		return fmt.Errorf("cari sesi: %w", err)
	}
	if err := s.q.RevokeSessionFamily(ctx, sqlc.RevokeSessionFamilyParams{FamilyID: sess.FamilyID, RevokedAt: &now}); err != nil {
		return fmt.Errorf("cabut sesi: %w", err)
	}
	return nil
}

func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	now := s.clock.Now()
	if err := s.q.RevokeUserSessions(ctx, sqlc.RevokeUserSessionsParams{UserID: userID, RevokedAt: &now}); err != nil {
		return fmt.Errorf("cabut semua sesi: %w", err)
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, familyID uuid.UUID, current, next, ip string) error {
	user, err := s.q.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("cari pengguna: %w", err)
	}
	ok, err := crypto.VerifyPassword(user.PasswordHash, current)
	if err != nil {
		return fmt.Errorf("verifikasi kata sandi: %w", err)
	}
	if !ok {
		return ErrInvalidCredentials
	}
	hash, err := crypto.HashPassword(next)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	return db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{ID: userID, PasswordHash: hash}); err != nil {
			return fmt.Errorf("ubah kata sandi: %w", err)
		}
		if err := q.RevokeUserSessionsExceptFamily(ctx, sqlc.RevokeUserSessionsExceptFamilyParams{UserID: userID, FamilyID: familyID, RevokedAt: &now}); err != nil {
			return fmt.Errorf("cabut sesi lain: %w", err)
		}
		return audit.Record(ctx, q, audit.Entry{DepotID: user.DepotID, UserID: &userID, Action: audit.ActionPasswordChanged, EntityType: "user", EntityID: &userID, IP: ip})
	})
}

func (s *Service) Me(ctx context.Context, userID uuid.UUID) (sqlc.User, sqlc.Depot, error) {
	user, err := s.q.GetUser(ctx, userID)
	if err != nil {
		return sqlc.User{}, sqlc.Depot{}, fmt.Errorf("cari pengguna: %w", err)
	}
	depot, err := s.q.GetDepot(ctx, user.DepotID)
	if err != nil {
		return sqlc.User{}, sqlc.Depot{}, fmt.Errorf("cari depot: %w", err)
	}
	return user, depot, nil
}
