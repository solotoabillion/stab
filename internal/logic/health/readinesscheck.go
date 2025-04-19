package health

import (
	"context"

	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type ReadinessCheckLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReadinessCheckLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadinessCheckLogic {
	return &ReadinessCheckLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReadinessCheckLogic) GetReadinessCheck(c echo.Context) (resp *types.Response, err error) {
	return &types.Response{
		Success: true,
		Message: "OK",
	}, nil
}
