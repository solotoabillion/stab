package settings

import (
	"context"

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSettingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateSettingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSettingLogic {
	return &CreateSettingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateSettingLogic) PostCreateSetting(c echo.Context, req *types.Setting) (resp *types.Response, err error) {
	setting := models.Setting{
		Category:    req.Category,
		Key:         req.Key,
		Value:       req.Value,
		DataType:    req.DataType,
		SortBy:      req.SortBy,
		Visibility:  req.Visibility,
		Description: req.Description,
	}
	if err := models.CreateSetting(l.svcCtx.DB, setting); err != nil {
		return &types.Response{
			Success: false,
			Message: "Failed to create setting: " + err.Error(),
		}, nil
	}
	return &types.Response{
		Success: true,
		Message: "Setting created successfully.",
	}, nil
}
