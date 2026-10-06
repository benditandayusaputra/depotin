package httpx

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type localKey int

const (
	localRequestID localKey = iota
	localEdgeTrusted
	localUserID
	localDepotID
	localRole
	localSessionFamily
)

func RequestID(c fiber.Ctx) string {
	if v, ok := c.Locals(localRequestID).(string); ok {
		return v
	}
	return ""
}

func EdgeTrusted(c fiber.Ctx) bool {
	v, ok := c.Locals(localEdgeTrusted).(bool)
	return ok && v
}

type Principal struct {
	UserID        uuid.UUID
	DepotID       uuid.UUID
	Role          string
	SessionFamily uuid.UUID
}

func SetPrincipal(c fiber.Ctx, p Principal) {
	c.Locals(localUserID, p.UserID)
	c.Locals(localDepotID, p.DepotID)
	c.Locals(localRole, p.Role)
	c.Locals(localSessionFamily, p.SessionFamily)
}

func CurrentPrincipal(c fiber.Ctx) (Principal, bool) {
	userID, ok := c.Locals(localUserID).(uuid.UUID)
	if !ok {
		return Principal{}, false
	}
	depotID, _ := c.Locals(localDepotID).(uuid.UUID)
	role, _ := c.Locals(localRole).(string)
	family, _ := c.Locals(localSessionFamily).(uuid.UUID)
	return Principal{UserID: userID, DepotID: depotID, Role: role, SessionFamily: family}, true
}

func ClientIP(c fiber.Ctx) string {
	if EdgeTrusted(c) {
		if ip := c.Get("X-Client-IP"); ip != "" {
			return ip
		}
	}
	return c.IP()
}
