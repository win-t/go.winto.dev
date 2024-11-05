package defmiddleware

import (
	"context"
	"net/http"
	"strings"
)

func BearerHeaderAuth(verifyFn func(ctx context.Context, token string) bool) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !verifyFn(ctx, token) {
				w.Header().Set("WWW-Authenticate", `Bearer realm="Restricted"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}
}
