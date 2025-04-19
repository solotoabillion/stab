package settings

import (
	"context"

	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateSettingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateSettingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSettingLogic {
	return &UpdateSettingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateSettingLogic) PutUpdateSetting(c echo.Context, req *types.SettingRequest) (resp *types.Response, err error) {
	setting := models.Setting{
		Category:    req.Category,
		Key:         req.Key,
		Value:       req.Value,
		DataType:    req.DataType,
		SortBy:      req.SortBy,
		Visibility:  req.Visibility,
		Description: req.Description,
	}
	if err := models.UpdateSetting(l.svcCtx.DB, setting); err != nil {
		return &types.Response{
			Success: false,
			Message: "Failed to update setting: " + err.Error(),
		}, nil
	}
	return &types.Response{
		Success: true,
		Message: "Setting updated successfully.",
	}, nil
}
