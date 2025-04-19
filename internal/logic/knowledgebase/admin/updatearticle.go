package admin

import (
	"context"

	"encoding/json"

	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/google/uuid"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateArticleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateArticleLogic {
	return &UpdateArticleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateArticleLogic) PutUpdateArticle(c echo.Context, req *types.KnowledgeBaseArticleRequest) (resp *types.KnowledgeBaseArticleResponse, err error) {
	id, err := uuid.Parse(req.ID)
	if err != nil {
		return &types.KnowledgeBaseArticleResponse{Success: false, Message: "Invalid article ID"}, err
	}
	var article models.KnowledgeBaseArticle
	if err := l.svcCtx.DB.First(&article, "id = ? AND is_active = ?", id, true).Error; err != nil {
		return &types.KnowledgeBaseArticleResponse{Success: false, Message: "Article not found"}, err
	}
	if req.Title != "" {
		article.Title = req.Title
	}
	if req.Body != "" {
		article.Body = req.Body
	}
	if req.Tags != nil {
		b, _ := json.Marshal(req.Tags)
		article.Tags = b
	}
	if req.Source != "" {
		article.Source = req.Source
	}
	if req.IsActive {
		article.IsActive = req.IsActive
	}
	if err := l.svcCtx.DB.Save(&article).Error; err != nil {
		return &types.KnowledgeBaseArticleResponse{Success: false, Message: "Failed to update article: " + err.Error()}, err
	}
	resp = &types.KnowledgeBaseArticleResponse{
		Success: true,
		Message: "Article updated successfully.",
		Article: types.MapKBModelToType(&article),
	}
	return
}
