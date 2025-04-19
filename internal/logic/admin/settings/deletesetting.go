package settings

import (
	"context"

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteSettingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteSettingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSettingLogic {
	return &DeleteSettingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteSettingLogic) DeleteSetting(c echo.Context, req *types.SettingRequest) (resp *types.Response, err error) {
	if err := models.DeleteSetting(l.svcCtx.DB, req.Category, req.Key); err != nil {
		return &types.Response{
			Success: false,
			Message: "Failed to delete setting: " + err.Error(),
		}, nil
	}
	return &types.Response{
		Success: true,
		Message: "Setting deleted successfully.",
	}, nil
}
