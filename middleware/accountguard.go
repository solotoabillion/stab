package middleware

import (
	"fmt"

	"stab/config"
	"stab/core/security"
	"stab/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm" // Added GORM import
)

type AccountGuardMiddleware struct {
	cfg *config.Config
	db  *gorm.DB // Changed from models.DB
}

// NewAccountGuardMiddleware creates a new AccountGuardMiddleware instance.
// It now accepts a *gorm.DB connection instead of models.DB.
func NewAccountGuardMiddleware(cfg *config.Config, db *gorm.DB) *AccountGuardMiddleware {
	return &AccountGuardMiddleware{
		cfg: cfg,
		db:  db, // Store the GORM DB instance
	}
}

func (m *AccountGuardMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tokenCookie, err := c.Request().Cookie(m.cfg.Auth.AccountCookieName)
		if err != nil {
			// fmt.Println("tokenCookie:", err)
			return next(c)
		}

		token := tokenCookie.Value

		unverifiedClaims, err := security.ParseUnverifiedJWT(token)
		if err != nil {
			fmt.Println("ParseUnverifiedJWT:", err)
			return next(c)
		}

		// check required claims
		id, ok := unverifiedClaims["id"].(string)
		if !ok {
			fmt.Println("Error: 'id' claim is not a string")
			return next(c)
		}

		// find account by id
		// find account by id using the new FindTeamByID function
		accountID, err := uuid.Parse(id)
		if err != nil {
			fmt.Println("Error parsing account ID from token:", err)
			return next(c)
		}
		account, err := models.FindTeamByID(m.db, accountID) // Use the GORM DB connection
		if err != nil {
			// fmt.Println(err)
			return next(c)
		}

		// verify token signature
		if _, err := security.ParseJWT(token, m.cfg.Auth.AccessSecret); err != nil {
			// fmt.Println(err)
			return next(c)
		}
		c.Set(ContextAccountKey, account)

		// fmt.Println("auth middleware: token verified")
		return next(c)
	}
}
