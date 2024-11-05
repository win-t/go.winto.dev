package defmiddleware

import (
	"context"
	"net/http"
)

func BasicAuth(verifyFn func(ctx context.Context, user, pass string) (ok bool, errmsg string)) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user, pass, _ := r.BasicAuth()
			ok, errmsg := verifyFn(ctx, user, pass)
			if !ok {
				if errmsg == "" {
					errmsg = "Unauthorized"
				}
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				http.Error(w, errmsg, http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}
}
