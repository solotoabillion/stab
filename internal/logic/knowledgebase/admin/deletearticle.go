package admin

import (
	"context"

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteArticleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteArticleLogic {
	return &DeleteArticleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteArticleLogic) DeleteArticle(c echo.Context, req *types.KnowledgeBaseArticleRequest) (resp *types.KnowledgeBaseArticleResponse, err error) {
	id, err := uuid.Parse(req.ID)
	if err != nil {
		return &types.KnowledgeBaseArticleResponse{Success: false, Message: "Invalid article ID"}, err
	}
	if err := models.SoftDeleteKnowledgeBaseArticle(l.svcCtx.DB, id); err != nil {
		return &types.KnowledgeBaseArticleResponse{Success: false, Message: "Failed to delete article: " + err.Error()}, err
	}
	resp = &types.KnowledgeBaseArticleResponse{
		Success: true,
		Message: "Article deleted successfully.",
	}
	return
}
