package middleware

import (
	"context"
	"net/http"

	"github.com/t11e/go-checkpoint"
)

type ctxKeyType int

const ctxKey ctxKeyType = iota

func SessionFromContext(ctx context.Context) (string, bool) {
	if v, ok := ctx.Value(ctxKey).(string); ok && v != "" {
		return v, true
	} else {
		return "", false
	}
}

func ContextWithSession(ctx context.Context, session string) context.Context {
	if session != "" {
		return context.WithValue(ctx, ctxKey, session)
	} else {
		return context.WithValue(ctx, ctxKey, nil)
	}
}

func New(client *checkpoint.Client) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		f := func(w http.ResponseWriter, req *http.Request) {
			if s, ok := checkpoint.SessionFromRequest(req); ok {
				next.ServeHTTP(w, req.WithContext(ContextWithSession(req.Context(), s)))
			} else {
				next.ServeHTTP(w, req)
			}
		}
		return http.HandlerFunc(f)
	}
}
