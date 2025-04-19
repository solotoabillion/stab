package tokens

import (
	"time"

	"stab/config"
	"stab/core/security"
	"stab/core/session"
	"stab/models"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

func SetAccountToken(c echo.Context, cfg *config.Config, membership *models.Membership) error {
	token, err := newAccountToken(cfg, membership)
	if err != nil {
		return err
	}

	session.SetCookie(cfg, c, token, cfg.Auth.AccountCookieName)
	return nil
}

// newAccountToken generates and returns a new auth record authentication token.
func newAccountToken(cfg *config.Config, membership *models.Membership) (string, error) {
	duration := time.Duration(cfg.Auth.AccessExpire) * time.Second
	expirationTime := time.Now().Add(duration)

	return security.NewJWT(
		jwt.MapClaims{
			"id":     membership.TeamID,
			"userID": membership.UserID,
		},
		(cfg.Auth.AccessSecret),
		expirationTime.Unix(),
	)
}
