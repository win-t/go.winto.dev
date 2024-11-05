package defresponse

import (
	"net/http"
)

// similar to [http.Redirect]
//
// [http.Redirect]: https://pkg.go.dev/net/http#Redirect
func Redirect(status int, url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, url, status)
	}
}
