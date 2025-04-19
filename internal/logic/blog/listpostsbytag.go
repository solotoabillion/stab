package blog

import (
	"context"

	"stab/models"
	"stab/svc"
	"stab/types"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListPostsByTagLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListPostsByTagLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPostsByTagLogic {
	return &ListPostsByTagLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListPostsByTagLogic) GetListPostsByTag(c echo.Context, req *types.BlogTagRequest) (resp *[]types.BlogPost, err error) {
	tag, err := models.FindTagBySlug(l.svcCtx.DB, req.TagSlug)
	if err != nil {
		l.Errorf("Failed to find tag by slug: %v", err)
		return nil, echo.NewHTTPError(404, "Tag not found")
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	perPage := req.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}

	offset := (page - 1) * perPage
	modelPosts, _, dbErr := models.FindPublishedBlogPostsByTag(l.svcCtx.DB, tag.ID.String(), perPage, offset)
	if dbErr != nil {
		l.Errorf("Failed to retrieve blog posts by tag: %v", dbErr)
		return nil, echo.NewHTTPError(500, "Failed to retrieve posts")
	}

	typePosts := make([]types.BlogPost, 0, len(modelPosts))
	for _, p := range modelPosts {
		post := types.BlogPost{
			ID:         p.ID.String(),
			Title:      p.Title,
			Slug:       p.Slug,
			Excerpt:    p.Excerpt,
			Content:    p.Content,
			Status:     string(p.Status),
			IsFeatured: p.IsFeatured,
			CoverImage: p.CoverImage,
			AuthorID:   "",
			ReadTime:   p.ReadTime,
			ViewCount:  p.ViewCount,
			LikeCount:  p.LikeCount,
			CreatedAt:  p.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  p.UpdatedAt.Format(time.RFC3339),
		}
		if p.PublishedAt != nil {
			post.PublishedAt = p.PublishedAt.Format(time.RFC3339)
		}
		if p.AuthorID != nil {
			post.AuthorID = p.AuthorID.String()
		}
		if len(p.Tags) > 0 {
			post.Tags = make([]types.Tag, len(p.Tags))
			for i, tag := range p.Tags {
				post.Tags[i] = types.Tag{
					ID:   tag.ID.String(),
					Name: tag.Name,
					Slug: tag.Slug,
				}
			}
		}
		if len(p.Categories) > 0 {
			post.Categories = make([]types.Category, len(p.Categories))
			for i, cat := range p.Categories {
				post.Categories[i] = types.Category{
					ID:   cat.ID.String(),
					Name: cat.Name,
					Slug: cat.Slug,
				}
			}
		}
		typePosts = append(typePosts, post)
	}
	return &typePosts, nil
}
