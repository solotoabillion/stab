package admin

import (
	"context"

	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type ReloadSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReloadSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReloadSettingsLogic {
	return &ReloadSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReloadSettingsLogic) PostReloadSettings(c echo.Context) (resp *types.Response, err error) {
	// todo: add your logic here and delete this line

	return
}
