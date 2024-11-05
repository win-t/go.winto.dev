package defresponse

import (
	"net/http"
)

// similar to [http.Error]
//
// [http.Error]: https://pkg.go.dev/net/http#Error
func Error(status int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, message, status)
	}
}
