package profile

import (
	"context"

	"github.com/solotoabillion/stab/core/session"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetEmailPreferencesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetEmailPreferencesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEmailPreferencesLogic {
	return &GetEmailPreferencesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetEmailPreferencesLogic) GetEmailPreferences(c echo.Context) (resp *types.EmailPreferences, err error) {
	user := session.UserFromContext(c)
	settings, err := user.GetSettings()
	if err != nil {
		return nil, echo.NewHTTPError(500, "Failed to load settings")
	}
	return &types.EmailPreferences{
		Marketing: settings.MarketingEmailsEnabled,
		Updates:   settings.SecurityAlertsEnabled,
		Security:  settings.SecurityAlertsEnabled,
	}, nil
}
