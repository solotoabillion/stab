package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/solotoabillion/stab/core/security"
	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

// GoogleUserInfo struct for parsing Google's response
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

type GoogleCallbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGoogleCallbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GoogleCallbackLogic {
	return &GoogleCallbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GoogleCallbackLogic) GetGoogleCallback(c echo.Context) (resp *types.GoogleResponse, err error) {
	// 1. Initialize OAuth config
	clientID := os.Getenv("GOOGLE_OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_OAUTH_REDIRECT_URL")

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		l.Error("Google OAuth environment variables not fully set")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Google OAuth not configured")
	}

	googleOAuthConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	// 2. Verify state token
	receivedState := c.QueryParam("state")
	expectedState := l.svcCtx.Config.Auth.GoogleOAuthStateString
	if receivedState != expectedState {
		l.Errorf("Invalid Google OAuth state token. Received: %s, Expected: %s", receivedState, expectedState)
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid state token")
	}

	// 3. Exchange authorization code for token
	code := c.QueryParam("code")
	token, err := googleOAuthConfig.Exchange(l.ctx, code)
	if err != nil {
		l.Errorf("Failed to exchange Google auth code for token: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to exchange code")
	}

	// 4. Fetch user info from Google
	client := googleOAuthConfig.Client(l.ctx, token)
	gogoleResponse, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		l.Errorf("Failed to get user info from Google: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get user info")
	}
	defer gogoleResponse.Body.Close()

	body, err := io.ReadAll(gogoleResponse.Body)
	if err != nil {
		l.Errorf("Failed to read Google user info response body: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to read user info")
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		l.Errorf("Failed to unmarshal Google user info: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse user info")
	}

	if !userInfo.VerifiedEmail {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Google email not verified")
	}

	// 5. Find or Create User in DB
	var user models.User
	err = l.svcCtx.DB.Where("email = ?", userInfo.Email).First(&user).Error

	if err != nil && err == gorm.ErrRecordNotFound {
		// User not found, create new user
		l.Infof("Google user %s not found, creating new user.", userInfo.Email)

		// Generate API key
		apiKey, err := models.GenerateAPIKey()
		if err != nil {
			l.Error("Error generating API key for Google user:", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Could not process registration")
		}

		// Create new user
		user = models.User{
			Email:            userInfo.Email,
			Role:             models.SystemRoleUser,
			ApiKey:           apiKey,
			DefaultSubdomain: generateRandomSubdomain(),
		}

		if err := l.svcCtx.DB.Create(&user).Error; err != nil {
			l.Errorf("Failed to create new user from Google OAuth (%s): %v", userInfo.Email, err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to create user account")
		}
	} else if err != nil {
		l.Errorf("Database error finding user %s from Google OAuth: %v", userInfo.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Database error during login")
	}

	// 6. Generate JWT for the user
	claims := jwt.MapClaims{
		"id":    user.ID.String(), // Use string representation of UUID
		"email": user.Email,
		"role":  user.Role,
		// Add other claims as needed, e.g., name
		// "name": user.Name,
	}
	jwtToken, err := security.NewJWT(claims, l.svcCtx.Config.Auth.AccessSecret, l.svcCtx.Config.Auth.AccessExpire)
	if err != nil {
		l.Errorf("Failed to generate JWT for Google user %s: %v", userInfo.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate session token")
	}

	// 7. Redirect back to frontend with token
	frontendURL := l.svcCtx.Config.FrontendURL
	if frontendURL == "" {
		frontendURL = "http://localhost:5173" // Default for local dev
	}

	redirectTarget := fmt.Sprintf("%s/app#token=%s&user=%s", frontendURL, jwtToken, url.QueryEscape(userInfo.Email))

	l.Infof("Google OAuth successful for %s. Redirecting.", userInfo.Email)
	return &types.GoogleResponse{
		Success:     true,
		Message:     "Successfully authenticated with Google",
		RedirectURL: redirectTarget,
	}, nil
}

// generateRandomSubdomain creates a memorable subdomain
func generateRandomSubdomain() string {
	// Simple implementation - in production, should use the full word lists and uniqueness check
	return fmt.Sprintf("user-%s", uuid.New().String()[:8])
}
