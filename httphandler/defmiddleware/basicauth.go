package defmiddleware

import (
	"context"
	"net/http"
)

func BasicAuth(verifyFn func(ctx context.Context, user, pass string) bool) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user, pass, _ := r.BasicAuth()
			if !verifyFn(ctx, user, pass) {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}
}
