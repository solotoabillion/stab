package profile

import (
	"context"

	"stab/core/session"
	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetSecuritySettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSecuritySettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSecuritySettingsLogic {
	return &GetSecuritySettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSecuritySettingsLogic) GetSecuritySettings(c echo.Context) (resp *types.SecuritySettings, err error) {
	user := session.UserFromContext(c)
	settings, _ := user.GetSettings()
	return &types.SecuritySettings{
		TwoFactorEnabled:   settings.TwoFactorEnabled,
		LastPasswordChange: "", // Add logic if you track this
	}, nil
}
