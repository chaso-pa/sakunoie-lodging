package handlers

import (
	"context"
	"io"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
)

// PostHandlers holds the post endpoints, all behind RequireAuth.
type PostHandlers struct {
	posts *services.PostService
}

func NewPostHandlers(posts *services.PostService) *PostHandlers {
	return &PostHandlers{posts: posts}
}

func (h *PostHandlers) ListPosts(ctx context.Context, input *models.ListPostsInput) (*models.ListPostsOutput, error) {
	posts, total, err := h.posts.List(ctx, middlewares.GetMemberID(ctx), services.PostFilter{
		Area:    input.Area,
		Weather: input.Weather,
		Month:   input.Month,
		Query:   input.Query,
		Sort:    input.Sort,
		Limit:   input.Limit,
		Offset:  input.Offset,
	})
	if err != nil {
		return nil, serviceError(ctx, "list posts", err)
	}
	resp := &models.ListPostsOutput{}
	resp.Body.Posts = posts
	resp.Body.Total = int(total)
	return resp, nil
}

func (h *PostHandlers) GetPost(ctx context.Context, input *models.PostIDInput) (*models.PostOutput, error) {
	post, err := h.posts.Get(ctx, middlewares.GetMemberID(ctx), input.ID)
	if err != nil {
		return nil, serviceError(ctx, "get post", err)
	}
	resp := &models.PostOutput{}
	resp.Body.Post = *post
	return resp, nil
}

func (h *PostHandlers) CreatePost(ctx context.Context, input *models.CreatePostInput) (*models.PostOutput, error) {
	// One photo per post: "media" is a single huma.FormFile, and Huma
	// answers 422 when a request carries more than one part of that name.
	data := input.RawBody.Data()
	var media io.Reader
	if data.Media.IsSet {
		defer data.Media.Close()
		media = data.Media
	}
	post, err := h.posts.Create(ctx, middlewares.GetMemberID(ctx), services.CreatePostParams{
		Caption:     data.Caption,
		SpotID:      data.SpotID,
		Language:    data.Language,
		Weather:     data.Weather,
		Temperature: data.Temperature,
		VisitedAt:   data.VisitedAt,
		Area:        data.Area,
		Tags:        data.Tags,
		Media:       media,
	})
	if err != nil {
		return nil, serviceError(ctx, "create post", err)
	}
	resp := &models.PostOutput{}
	resp.Body.Post = *post
	return resp, nil
}

func (h *PostHandlers) DeletePost(ctx context.Context, input *models.PostIDInput) (*models.OKOutput, error) {
	if err := h.posts.Delete(ctx, middlewares.GetMemberID(ctx), input.ID); err != nil {
		return nil, serviceError(ctx, "delete post", err)
	}
	return models.NewOKOutput(), nil
}

func (h *PostHandlers) ToggleLike(ctx context.Context, input *models.PostIDInput) (*models.ToggleLikeOutput, error) {
	liked, count, err := h.posts.ToggleLike(ctx, middlewares.GetMemberID(ctx), input.ID)
	if err != nil {
		return nil, serviceError(ctx, "toggle like", err)
	}
	resp := &models.ToggleLikeOutput{}
	resp.Body.Liked = liked
	resp.Body.LikeCount = count
	return resp, nil
}

func (h *PostHandlers) ToggleBookmark(ctx context.Context, input *models.PostIDInput) (*models.ToggleBookmarkOutput, error) {
	bookmarked, err := h.posts.ToggleBookmark(ctx, middlewares.GetMemberID(ctx), input.ID)
	if err != nil {
		return nil, serviceError(ctx, "toggle bookmark", err)
	}
	resp := &models.ToggleBookmarkOutput{}
	resp.Body.Bookmarked = bookmarked
	return resp, nil
}

func (h *PostHandlers) ListBookmarkedPosts(ctx context.Context, _ *struct{}) (*models.PostListOutput, error) {
	posts, err := h.posts.Bookmarked(ctx, middlewares.GetMemberID(ctx))
	if err != nil {
		return nil, serviceError(ctx, "list bookmarked posts", err)
	}
	resp := &models.PostListOutput{}
	resp.Body.Posts = posts
	return resp, nil
}

func (h *PostHandlers) ListMyPosts(ctx context.Context, _ *struct{}) (*models.PostListOutput, error) {
	posts, err := h.posts.Mine(ctx, middlewares.GetMemberID(ctx))
	if err != nil {
		return nil, serviceError(ctx, "list own posts", err)
	}
	resp := &models.PostListOutput{}
	resp.Body.Posts = posts
	return resp, nil
}

func (h *PostHandlers) GetRankings(ctx context.Context, input *models.RankingsInput) (*models.RankingsOutput, error) {
	var (
		items []models.RankingItem
		err   error
	)
	if input.Type == "posts" {
		items, err = h.posts.PostRanking(ctx, middlewares.GetMemberID(ctx), input.Limit)
	} else {
		items, err = h.posts.SpotRanking(ctx, middlewares.GetMemberID(ctx), input.Limit)
	}
	if err != nil {
		return nil, serviceError(ctx, "get rankings", err)
	}
	resp := &models.RankingsOutput{}
	resp.Body.Items = items
	return resp, nil
}
