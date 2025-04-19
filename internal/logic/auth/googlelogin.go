package auth

import (
	"context"
	"net/http"

	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleLoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGoogleLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GoogleLoginLogic {
	return &GoogleLoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GoogleLoginLogic) GetGoogleLogin(c echo.Context) (resp *types.GoogleResponse, err error) {
	// 1. Initialize OAuth config if not already done
	clientID := l.svcCtx.Config.Auth.GoogleOAuthClientID
	clientSecret := l.svcCtx.Config.Auth.GoogleOAuthClientSecret
	redirectURL := l.svcCtx.Config.Auth.GoogleOAuthRedirectURL

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

	// 2. Generate state token (should be stored in user's session/Redis in production)
	state := l.svcCtx.Config.Auth.GoogleOAuthStateString
	if state == "" {
		l.Error("Google OAuth state string not configured")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "OAuth state not configured")
	}

	// 3. Get authorization URL
	redirectURL = googleOAuthConfig.AuthCodeURL(state)

	l.Infof("Generated Google OAuth URL for login: %s", redirectURL)

	// 4. Return the URL in the response
	resp = &types.GoogleResponse{
		Success:     true,
		Message:     "Successfully generated Google OAuth URL",
		RedirectURL: redirectURL,
	}

	return resp, nil
}
