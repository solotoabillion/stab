package profile

import (
	"context"

	"stab/core/session"
	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type EmailPreferencesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEmailPreferencesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EmailPreferencesLogic {
	return &EmailPreferencesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EmailPreferencesLogic) GetEmailPreferences(c echo.Context) (resp *types.EmailPreferences, err error) {
	user := session.UserFromContext(c)
	settings, _ := user.GetSettings()
	return &types.EmailPreferences{
		Marketing: settings.MarketingEmailsEnabled,
		Updates:   settings.SecurityAlertsEnabled,
		Security:  settings.SecurityAlertsEnabled,
	}, nil
}
