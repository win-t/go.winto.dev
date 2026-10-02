package defmiddleware

import "net/http"

// run the handler using runner, runner must catch any panic and return it as error, the errHandler is used for logging that error, errHandler must not panic.
func ErrorHandling(code int, msg string, runner func(func()) error, logErr func(error)) func(http.HandlerFunc) http.HandlerFunc {
	return func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if err := runner(func() { handler(w, r) }); err != nil {
				logErr(err)
				w.Write([]byte(msg))
				w.WriteHeader(code)
				_ = http.NewResponseController(w).Flush()
				panic(http.ErrAbortHandler)
			}
		}
	}
}
