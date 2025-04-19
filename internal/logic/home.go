package logic

import (
	"context"

	"stab/svc"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type HomeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHomeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HomeLogic {
	return &HomeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HomeLogic) GetHome(c echo.Context) {
	// todo: add your logic here and delete this line

}
