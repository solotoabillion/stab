package settings

import (
	"context"

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSettingsLogic {
	return &ListSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListSettingsLogic) GetListSettings(c echo.Context) (resp *[]types.Setting, err error) {
	categoryPrefix := c.QueryParam("category")
	var settings []models.Setting
	if categoryPrefix != "" {
		settings, err = models.FindSettingsByCategoryPrefix(l.svcCtx.DB, categoryPrefix)
	} else {
		settings, err = models.FindAllSettings(l.svcCtx.DB)
	}
	if err != nil {
		return nil, err
	}
	result := make([]types.Setting, len(settings))
	for i, s := range settings {
		result[i] = types.Setting{
			Category:    s.Category,
			Key:         s.Key,
			Value:       s.Value,
			DataType:    s.DataType,
			SortBy:      s.SortBy,
			Visibility:  s.Visibility,
			Description: s.Description,
		}
	}
	return &result, nil
}
