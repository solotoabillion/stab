package settings

import (
	"context"

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetSettingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSettingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSettingLogic {
	return &GetSettingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSettingLogic) GetSetting(c echo.Context, req *types.SettingRequest) (resp *types.Setting, err error) {
	setting, err := models.GetSetting(l.svcCtx.DB, req.Category, req.Key)
	if err != nil {
		return nil, err
	}
	return &types.Setting{
		Category:    setting.Category,
		Key:         setting.Key,
		Value:       setting.Value,
		DataType:    setting.DataType,
		SortBy:      setting.SortBy,
		Visibility:  setting.Visibility,
		Description: setting.Description,
	}, nil
}
