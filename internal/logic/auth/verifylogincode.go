package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/solotoabillion/stab/core/security"
	"github.com/solotoabillion/stab/core/session"
	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type VerifyLoginCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewVerifyLoginCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyLoginCodeLogic {
	return &VerifyLoginCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VerifyLoginCodeLogic) PostVerifyLoginCode(c echo.Context, req *types.VerifyCodeRequest) (resp *types.Response, err error) {
	l.Logger.Infof("VerifyLoginCode attempt for email: %s", req.Email)

	// TODO: Implement actual code verification.
	// Currently bypassed because RequestLoginCode migration skipped storing the code (e.g., in Redis).
	// This handler currently acts as a direct login mechanism based on email only if the user exists.
	l.Logger.Errorf("Login code verification is currently BYPASSED for email: %s", req.Email) // Changed Warnf to Errorf

	// 1. Find the user by email (since code verification is skipped)
	// 1. Find the user by email using model function (since code verification is skipped)
	userPtr, err := models.FindUserByEmail(l.svcCtx.DB, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Infof("VerifyLoginCode failed for email %s: user not found", req.Email)
			return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid email or code.")
		}
		l.Errorf("Database error during VerifyLoginCode user lookup for %s: %v", req.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Database error during login verification")
	}
	user := *userPtr // Dereference if found

	// 2. User found, generate JWT (similar to LoginUser)
	claims := jwt.MapClaims{
		"id":    user.ID.String(),
		"email": user.Email,
		"role":  user.Role,
	}
	tokenString, err := security.NewJWT(claims, l.svcCtx.Config.Auth.AccessSecret, l.svcCtx.Config.Auth.AccessExpire)
	if err != nil {
		l.Errorf("Error generating JWT for user %s during VerifyLoginCode: %v", user.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Could not process login (token generation)")
	}

	l.Infof("Passwordless login successful (verification bypassed) for: %s", user.Email)

	// 3. Set auth cookie (similar to LoginUser)
	session.SetCookie(l.svcCtx.Config, c, tokenString, l.svcCtx.Config.Auth.UserCookieName)

	// 4. Return standard success response
	resp = &types.Response{
		Success: true,
		Message: "Login successful", // Keep message consistent with password login
	}

	return resp, nil
}
