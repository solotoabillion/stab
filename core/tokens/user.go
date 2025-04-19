package tokens

import (
	"encoding/json" // Added for JSON handling
	"time"

	"stab/config"
	"stab/core/security"
	"stab/core/session"
	"stab/models"
	"stab/types" // Added for UserProfileData type

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

func SetUserToken(c echo.Context, cfg *config.Config, user *models.User) error {
	token, err := newUserToken(cfg, user)
	if err != nil {
		return err
	}

	session.SetCookie(cfg, c, token, cfg.Auth.UserCookieName)
	return nil
}

// newUserToken generates and returns a new user authentication token.
func newUserToken(cfg *config.Config, user *models.User) (string, error) {
	duration := time.Duration(cfg.Auth.AccessExpire) * time.Second
	expirationTime := time.Now().Add(duration)

	// Unmarshal profile data to get names
	var profileData types.UserProfileData
	if user.ProfileData != nil {
		// Log error but proceed? Or return error? Let's proceed with empty names if unmarshal fails.
		_ = json.Unmarshal(user.ProfileData, &profileData)
		// Consider adding error logging here if needed: logx.Errorf(...)
	}

	return security.NewJWT(
		jwt.MapClaims{
			"id":         user.ID.String(),      // Ensure ID is string
			"first_name": profileData.FirstName, // Use data from unmarshalled struct
			"last_name":  profileData.LastName,  // Use data from unmarshalled struct
			"email":      user.Email,
			// Add other claims as needed, e.g., role?
			// "role": string(user.Role),
		},
		cfg.Auth.AccessSecret,
		expirationTime.Unix(),
	)
}
