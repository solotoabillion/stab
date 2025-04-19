package knowledgebase

import (
	"context"

	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/solotoabillion/stab/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetArticleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetArticleLogic {
	return &GetArticleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetArticleLogic) GetArticle(c echo.Context, req *types.KnowledgeBaseArticleRequest) (resp *types.KnowledgeBaseArticleResponse, err error) {
	id, err := uuid.Parse(req.ID)
	if err != nil {
		return &types.KnowledgeBaseArticleResponse{Success: false, Message: "Invalid article ID"}, err
	}
	var article models.KnowledgeBaseArticle
	if err := l.svcCtx.DB.First(&article, "id = ? AND is_active = ?", id, true).Error; err != nil {
		return &types.KnowledgeBaseArticleResponse{Success: false, Message: "Article not found"}, err
	}
	resp = &types.KnowledgeBaseArticleResponse{
		Success: true,
		Message: "Article fetched successfully.",
		Article: types.MapKBModelToType(&article),
	}
	return
}
