package defmiddleware

import (
	"context"
	"net/http"
	"strings"
)

func BearerHeaderAuth(verifyFn func(ctx context.Context, token string) (ok bool, errmsg string)) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			ok, errmsg := verifyFn(ctx, token)
			if !ok {
				if errmsg == "" {
					errmsg = "Unauthorized"
				}
				w.Header().Set("WWW-Authenticate", `Bearer realm="Restricted"`)
				http.Error(w, errmsg, http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}
}
