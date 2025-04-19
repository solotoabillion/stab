package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/solotoabillion/stab/core/security" // Added session import for cookie setting
	"github.com/solotoabillion/stab/core/session"
	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Removed local LoginResponse struct definition

type LoginUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginUserLogic {
	return &LoginUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// extractProfileData extracts first and last name from the JSON profile data
func extractProfileData(profileData datatypes.JSON) (firstName, lastName string) {
	// Default empty strings
	firstName = ""
	lastName = ""

	// Try to unmarshal the JSON
	var data map[string]interface{}
	if err := json.Unmarshal(profileData, &data); err != nil {
		return
	}

	// Extract first and last name if they exist
	if fn, ok := data["firstName"].(string); ok {
		firstName = fn
	}
	if ln, ok := data["lastName"].(string); ok {
		lastName = ln
	}

	return
}

func (l *LoginUserLogic) PostLoginUser(c echo.Context, req *types.LoginRequest) (resp *types.LoginResponse, err error) { // Changed return type
	// 1. Find user by email using model function
	userPtr, err := models.FindUserByEmail(l.svcCtx.DB, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Infof("Login attempt failed for email %s: user not found", req.Email)
			return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
		}
		l.Errorf("Database error during login for %s: %v", req.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Database error during login")
	}
	user := *userPtr // Dereference if found

	// 2. Check password
	if !user.CheckPassword(req.Password) {
		l.Infof("Login attempt failed for email %s: invalid password", req.Email) // Use Infof instead of Warnf
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
	}

	// 3. Generate JWT
	claims := jwt.MapClaims{
		"id":    user.ID.String(), // Use string representation of UUID
		"email": user.Email,
		"role":  user.Role,
		// Add other claims as needed, e.g., name
		// "name": user.Name,
	}
	tokenString, err := security.NewJWT(claims, l.svcCtx.Config.Auth.AccessSecret, l.svcCtx.Config.Auth.AccessExpire)
	if err != nil {
		l.Errorf("Error generating JWT for user %s: %v", user.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Could not process login (token generation)")
	}

	l.Infof("User logged in successfully: %s", user.Email)

	// 4. Set auth cookie
	session.SetCookie(l.svcCtx.Config, c, tokenString, l.svcCtx.Config.Auth.UserCookieName)

	// 5. Return standard success response
	firstName, lastName := extractProfileData(user.ProfileData)
	resp = &types.LoginResponse{
		Success: true,
		Message: "Login successful",
		Token:   tokenString,
		User: types.User{
			ID:    user.ID.String(),
			Email: user.Email,
			ProfileData: types.UserProfileData{
				FirstName: firstName,
				LastName:  lastName,
			},
			Role:             string(user.Role),
			ApiKey:           user.ApiKey,
			DefaultSubdomain: user.DefaultSubdomain,
		},
	}

	return resp, nil // Return standard response and nil error
}
