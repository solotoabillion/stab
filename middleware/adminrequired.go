package middleware

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// AdminRequired checks if the authenticated user has admin privileges.
// Renamed to lowercase to match apparent soul generator convention/bug.
// Assumes JWT middleware runs before this and populates c.Get("user").
func AdminRequired(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userToken, ok := c.Get("user").(*jwt.Token) // Assumes JWT middleware uses "user" key
		if !ok || userToken == nil {
			c.Logger().Error("AdminRequired: JWT token missing or invalid type in context (key 'user')")
			return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized: Missing or invalid token context")
		}

		// Use JWTCustomClaims defined in middleware.go (or central location)
		claims, ok := userToken.Claims.(*JWTCustomClaims)
		if !ok || claims == nil {
			regClaims, okFallback := userToken.Claims.(*jwt.RegisteredClaims)
			if okFallback && regClaims != nil {
				c.Logger().Warnf("AdminRequired: Standard claims found, but Admin flag missing. Denying admin access for user (ID: %s) to %s", regClaims.Subject, c.Request().URL.Path)
				return echo.NewHTTPError(http.StatusForbidden, "Forbidden: Administrator access required (invalid claims type)")
			}
			c.Logger().Error("AdminRequired: Failed to cast JWT claims to JWTCustomClaims or RegisteredClaims")
			return echo.NewHTTPError(http.StatusForbidden, "Forbidden: Invalid claims format")
		}

		if !claims.Admin {
			c.Logger().Warnf("AdminRequired: Non-admin user (ID: %s, Name: %s) attempted admin access to %s", claims.Subject, claims.Name, c.Request().URL.Path)
			return echo.NewHTTPError(http.StatusForbidden, "Forbidden: Administrator access required")
		}
		return next(c)
	}
}
