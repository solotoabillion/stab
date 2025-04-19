package middleware

import (
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

const (
	ContextAccountKey string = "accountCtx"
	ContextUserKey    string = "userCtx"
)

// JWTCustomClaims defines the structure for JWT claims, including admin status.
// This should match the claims structure used by the JWT middleware configured via `jwt: Auth` in the .api file.
// TODO: Centralize this definition if possible (e.g., in types/types.go if soul generates it)
type JWTCustomClaims struct {
	Name  string `json:"name"`
	Admin bool   `json:"admin"`
	jwt.RegisteredClaims
}

// Note: adminrequired function moved to adminrequired.go

// InternalAPIAuth checks for a shared secret header for internal API calls.
// This function can be referenced in the .api file if needed.
func InternalAPIAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		internalAPISecret := os.Getenv("INTERNAL_API_SECRET") // Read from environment

		if internalAPISecret == "" {
			c.Logger().Error("CRITICAL: INTERNAL_API_SECRET environment variable is not set. Internal API is insecure.")
			// Decide whether to block or allow based on security needs
			// return echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error: Configuration missing")
		}

		providedSecret := c.Request().Header.Get("X-Internal-Secret")
		if internalAPISecret != "" && providedSecret != internalAPISecret {
			c.Logger().Warnf("WARN: Invalid or missing X-Internal-Secret header for internal API request from %s", c.RealIP())
			return echo.NewHTTPError(http.StatusForbidden, "Forbidden: Invalid internal secret")
		}
		return next(c)
	}
}

// Note: CustomStaticMiddleware and NewNoCacheMiddleware are assumed to be defined elsewhere
// or provided by the soul framework/dependencies, as they are used in servicecontext.go.
