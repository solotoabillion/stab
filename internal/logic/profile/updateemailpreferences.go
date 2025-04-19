package profile

import (
	"context"

	"stab/core/session"
	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateEmailPreferencesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateEmailPreferencesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateEmailPreferencesLogic {
	return &UpdateEmailPreferencesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateEmailPreferencesLogic) PutUpdateEmailPreferences(c echo.Context, req *types.EmailPreferences) (resp *types.Response, err error) {
	user := session.UserFromContext(c)
	settings, _ := user.GetSettings()
	settings.MarketingEmailsEnabled = req.Marketing
	settings.SecurityAlertsEnabled = req.Security || req.Updates
	if err := user.UpdateSettings(l.svcCtx.DB, settings); err != nil {
		return nil, echo.NewHTTPError(500, "Failed to update preferences")
	}
	return &types.Response{Success: true, Message: "Preferences updated"}, nil
}
