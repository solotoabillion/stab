package blog

import (
	"context"
	"time"

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetPostBySlugLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetPostBySlugLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPostBySlugLogic {
	return &GetPostBySlugLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPostBySlugLogic) GetPostBySlug(c echo.Context, req *types.BlogPostRequest) (resp *types.BlogPost, err error) {
	// Retrieve the post by slug using the models layer
	modelPost, dbErr := models.FindPublishedBlogPostBySlug(l.svcCtx.DB, req.Slug)
	if dbErr != nil {
		l.Errorf("Failed to retrieve blog post by slug: %v", dbErr)
		return nil, echo.NewHTTPError(404, "Post not found")
	}

	// Map model to API type
	post := &types.BlogPost{
		ID:         modelPost.ID.String(),
		Title:      modelPost.Title,
		Slug:       modelPost.Slug,
		Excerpt:    modelPost.Excerpt,
		Content:    modelPost.Content,
		Status:     string(modelPost.Status),
		IsFeatured: modelPost.IsFeatured,
		CoverImage: modelPost.CoverImage,
		AuthorID:   "",
		ReadTime:   modelPost.ReadTime,
		ViewCount:  modelPost.ViewCount,
		LikeCount:  modelPost.LikeCount,
		CreatedAt:  modelPost.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  modelPost.UpdatedAt.Format(time.RFC3339),
	}
	if modelPost.PublishedAt != nil {
		post.PublishedAt = modelPost.PublishedAt.Format(time.RFC3339)
	}
	if modelPost.AuthorID != nil {
		post.AuthorID = modelPost.AuthorID.String()
	}
	// Map Tags
	if len(modelPost.Tags) > 0 {
		post.Tags = make([]types.Tag, len(modelPost.Tags))
		for i, tag := range modelPost.Tags {
			post.Tags[i] = types.Tag{
				ID:   tag.ID.String(),
				Name: tag.Name,
				Slug: tag.Slug,
			}
		}
	}
	// Map Categories
	if len(modelPost.Categories) > 0 {
		post.Categories = make([]types.Category, len(modelPost.Categories))
		for i, cat := range modelPost.Categories {
			post.Categories[i] = types.Category{
				ID:   cat.ID.String(),
				Name: cat.Name,
				Slug: cat.Slug,
			}
		}
	}
	return post, nil
}
