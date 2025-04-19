package profile

import (
	"context"

	"stab/core/session"
	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteApiKeyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteApiKeyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteApiKeyLogic {
	return &DeleteApiKeyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteApiKeyLogic) DeleteApiKey(c echo.Context) (resp *types.Response, err error) {
	user := session.UserFromContext(c)
	if err := models.UpdateUserAPIKey(l.svcCtx.DB, user.ID, ""); err != nil {
		return nil, echo.NewHTTPError(500, "Failed to delete API key")
	}
	return &types.Response{Success: true, Message: "API key deleted"}, nil
}
