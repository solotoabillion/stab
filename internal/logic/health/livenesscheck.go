package health

import (
	"context"

	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type LivenessCheckLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLivenessCheckLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LivenessCheckLogic {
	return &LivenessCheckLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LivenessCheckLogic) GetLivenessCheck(c echo.Context) (resp *types.Response, err error) {
	return &types.Response{
		Success: true,
		Message: "OK",
	}, nil
}
