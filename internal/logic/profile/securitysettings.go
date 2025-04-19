package profile

import (
	"context"

	"github.com/solotoabillion/stab/core/session"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type SecuritySettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSecuritySettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SecuritySettingsLogic {
	return &SecuritySettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SecuritySettingsLogic) GetSecuritySettings(c echo.Context) (resp *types.SecuritySettings, err error) {
	user := session.UserFromContext(c)
	settings, _ := user.GetSettings()
	return &types.SecuritySettings{
		TwoFactorEnabled:   settings.TwoFactorEnabled,
		LastPasswordChange: "", // Add logic if you track this
	}, nil
}
