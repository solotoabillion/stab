package admin

import (
	"context"
	"encoding/json"

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/datatypes"
)

type CreateArticleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateArticleLogic {
	return &CreateArticleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateArticleLogic) PostCreateArticle(c echo.Context, req *types.KnowledgeBaseArticleRequest) (resp *types.KnowledgeBaseArticleResponse, err error) {
	article := &models.KnowledgeBaseArticle{
		Title:    req.Title,
		Body:     req.Body,
		Source:   req.Source,
		IsActive: req.IsActive,
	}
	if req.Tags != nil {
		b, _ := json.Marshal(req.Tags)
		article.Tags = datatypes.JSON(b)
	}
	if err := models.CreateKnowledgeBaseArticle(l.svcCtx.DB, article); err != nil {
		return &types.KnowledgeBaseArticleResponse{Success: false, Message: "Failed to create article: " + err.Error()}, err
	}
	resp = &types.KnowledgeBaseArticleResponse{
		Success: true,
		Message: "Article created successfully.",
		Article: types.MapKBModelToType(article),
	}
	return
}
