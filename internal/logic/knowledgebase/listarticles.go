package knowledgebase

import (
	"context"

	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListArticlesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListArticlesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListArticlesLogic {
	return &ListArticlesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListArticlesLogic) GetListArticles(c echo.Context) (resp *types.KnowledgeBaseArticlesResponse, err error) {
	var articles []models.KnowledgeBaseArticle
	if err := l.svcCtx.DB.Where("is_active = ?", true).Order("created_at desc").Find(&articles).Error; err != nil {
		return &types.KnowledgeBaseArticlesResponse{Success: false, Message: "Failed to list articles: " + err.Error()}, err
	}
	resp = &types.KnowledgeBaseArticlesResponse{
		Success:  true,
		Message:  "Articles listed successfully.",
		Articles: make([]types.KnowledgeBaseArticle, len(articles)),
	}
	for i, a := range articles {
		resp.Articles[i] = types.MapKBModelToType(&a)
	}
	return
}
