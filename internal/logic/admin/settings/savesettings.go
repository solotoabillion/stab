package settings

import (
	"context"

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type SaveSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSaveSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveSettingsLogic {
	return &SaveSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SaveSettingsLogic) PostSaveSettings(c echo.Context, req *[]types.Setting) (resp *types.Response, err error) {
	var settings []models.Setting
	for _, s := range *req {
		settings = append(settings, models.Setting{
			Category:    s.Category,
			Key:         s.Key,
			Value:       s.Value,
			DataType:    s.DataType,
			SortBy:      s.SortBy,
			Visibility:  s.Visibility,
			Description: s.Description,
		})
	}
	if err := models.SaveSettings(l.svcCtx.DB, settings); err != nil {
		return &types.Response{
			Success: false,
			Message: "Failed to save settings: " + err.Error(),
		}, nil
	}
	return &types.Response{
		Success: true,
		Message: "Settings saved successfully.",
	}, nil
}
