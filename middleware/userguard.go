package middleware

import (
	"fmt"

	"github.com/solotoabillion/stab/config"
	"github.com/solotoabillion/stab/core/security"
	"github.com/solotoabillion/stab/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm" // Added GORM import
)

type UserGuardMiddleware struct {
	cfg *config.Config
	db  *gorm.DB // Changed from models.DB
}

// NewUserGuardMiddleware creates a new UserGuardMiddleware instance.
// It now accepts a *gorm.DB connection instead of models.DB.
func NewUserGuardMiddleware(cfg *config.Config, db *gorm.DB) *UserGuardMiddleware {
	return &UserGuardMiddleware{
		cfg: cfg,
		db:  db, // Store the GORM DB instance
	}
}

func (m *UserGuardMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tokenCookie, err := c.Request().Cookie(m.cfg.Auth.UserCookieName)
		if err != nil {
			// fmt.Println("tokenCookie:", err)
			return next(c)
		}

		token := tokenCookie.Value

		unverifiedClaims, err := security.ParseUnverifiedJWT(token)
		if err != nil {
			// fmt.Println("ParseUnverifiedJWT:", err)
			return next(c)
		}

		id, ok := unverifiedClaims["id"].(string)
		if !ok {
			fmt.Println("Error: 'id' claim is not a string")
			return next(c)
		}

		// find user by id
		// find user by id using the new FindUserByID function
		userID, err := uuid.Parse(id)
		if err != nil {
			fmt.Println("Error parsing user ID from token:", err)
			return next(c)
		}
		user, err := models.FindUserByID(m.db, userID) // Use the GORM DB connection
		if err != nil {
			// fmt.Println(err)
			return next(c)
		}

		// verify token signature
		if _, err := security.ParseJWT(token, m.cfg.Auth.AccessSecret); err != nil {
			// fmt.Println(err)
			return next(c)
		}
		c.Set(ContextUserKey, user)

		// fmt.Println("auth middleware: token verified")
		return next(c)
	}
}
