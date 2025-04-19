package blog

import (
	"context"
	"net/http"
	"time"

	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListPublishedPostsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListPublishedPostsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPublishedPostsLogic {
	return &ListPublishedPostsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetListPublishedPosts retrieves a paginated list of published blog posts.
func (l *ListPublishedPostsLogic) GetListPublishedPosts(c echo.Context, req *types.BlogListRequest) (*types.BlogPostsResponse, error) {
	// 1. Validate and prepare query parameters
	page := req.Page
	if page < 1 {
		page = 1
	}
	perPage := req.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}

	// 2. Prepare query parameters
	params := models.BlogPostsQueryParams{
		Limit:    perPage,
		Offset:   (page - 1) * perPage,
		Sort:     req.Sort,
		Featured: req.Featured,
		AuthorID: req.AuthorID,
		Search:   req.Search,
	}

	// 3. Query Database using model function
	modelPosts, total, dbErr := models.FindPublishedBlogPosts(l.svcCtx.DB, params)
	if dbErr != nil {
		l.Errorf("Failed to retrieve published blog posts: %v", dbErr)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve posts")
	}

	// 4. Map models.BlogPost to types.BlogPost
	typePosts := make([]types.BlogPost, 0, len(modelPosts))
	for _, p := range modelPosts {
		post := types.BlogPost{
			ID:         p.ID.String(),
			Title:      p.Title,
			Slug:       p.Slug,
			Content:    p.Content,
			Excerpt:    p.Excerpt,
			Status:     string(p.Status),
			IsFeatured: p.IsFeatured,
			CoverImage: p.CoverImage,
			AuthorID:   p.AuthorID.String(),
			ReadTime:   p.ReadTime,
			ViewCount:  p.ViewCount,
			LikeCount:  p.LikeCount,
			CreatedAt:  p.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  p.UpdatedAt.Format(time.RFC3339),
		}

		// Handle optional PublishedAt
		if p.PublishedAt != nil {
			post.PublishedAt = p.PublishedAt.Format(time.RFC3339)
		}

		// Map Tags
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

		// Map Categories
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

	// 5. Prepare response
	resp := &types.BlogPostsResponse{
		Success: true,
		Message: "Posts retrieved successfully",
		Posts:   typePosts,
		Total:   int(total),
		Page:    page,
		PerPage: perPage,
	}

	l.Infof("Retrieved %d published blog posts (page %d, perPage %d, total %d)", len(typePosts), page, perPage, total)
	return resp, nil
}
