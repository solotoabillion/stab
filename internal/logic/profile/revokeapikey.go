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

type RevokeApiKeyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRevokeApiKeyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeApiKeyLogic {
	return &RevokeApiKeyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RevokeApiKeyLogic) PostRevokeApiKey(c echo.Context) (resp *types.ApiKeyResponse, err error) {
	user := session.UserFromContext(c)
	if err := models.UpdateUserAPIKey(l.svcCtx.DB, user.ID, ""); err != nil {
		return nil, echo.NewHTTPError(500, "Failed to revoke API key")
	}
	return &types.ApiKeyResponse{ApiKey: ""}, nil
}
