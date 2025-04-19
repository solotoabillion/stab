package knowledgebase

import (
	"context"

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type SearchArticlesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchArticlesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchArticlesLogic {
	return &SearchArticlesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchArticlesLogic) GetSearchArticles(c echo.Context, req *types.KnowledgeBaseSearchRequest) (resp *types.KnowledgeBaseArticlesResponse, err error) {
	articles, err := models.SearchKnowledgeBaseArticles(l.svcCtx.DB, req.Query)
	if err != nil {
		return &types.KnowledgeBaseArticlesResponse{Success: false, Message: "Failed to search articles: " + err.Error()}, err
	}
	resp = &types.KnowledgeBaseArticlesResponse{
		Success:  true,
		Message:  "Articles found successfully.",
		Articles: make([]types.KnowledgeBaseArticle, len(articles)),
	}
	for i, a := range articles {
		resp.Articles[i] = types.MapKBModelToType(&a)
	}
	return
}
