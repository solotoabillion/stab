package health

import (
	"context"

	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type HealthCheckLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHealthCheckLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HealthCheckLogic {
	return &HealthCheckLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HealthCheckLogic) GetHealthCheck(c echo.Context) (resp *types.Response, err error) {
	return &types.Response{
		Success: true,
		Message: "OK",
	}, nil
}
