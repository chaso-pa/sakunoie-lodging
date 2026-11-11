package middlewares

import (
	"context"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

type editTokenKey struct{}

// EditToken is a Huma middleware for /manage operations: it sets
// Cache-Control: no-store and puts the bearer token (possibly "") in the
// context. The service rejects a missing or bad token with 401.
func EditToken(ctx huma.Context, next func(huma.Context)) {
	ctx.SetHeader("Cache-Control", "no-store")
	next(huma.WithValue(ctx, editTokenKey{}, bearerToken(ctx)))
}

// bearerToken returns the `Authorization: Bearer` credential, or "".
func bearerToken(ctx huma.Context) string {
	if h := ctx.Header("Authorization"); len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// EditTokenFrom returns the bearer token placed by EditToken.
func EditTokenFrom(ctx context.Context) string {
	t, _ := ctx.Value(editTokenKey{}).(string)
	return t
}

// NoStore is a Huma middleware that disables caching (create responses
// carry the one-time edit token).
func NoStore(ctx huma.Context, next func(huma.Context)) {
	ctx.SetHeader("Cache-Control", "no-store")
	next(ctx)
}
