package httphandler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"go.winto.dev/httphandler"
	"go.winto.dev/httphandler/defresponse"
)

func ExampleOf() {
	mux := httphandler.Of(func(r *http.Request) http.HandlerFunc {
		return defresponse.Text(200, "Hello")
	})

	req := httptest.NewRequest("GET", "/", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	fmt.Println(res.Body.String())
	// Output: Hello
}

func ExampleMergeHeader() {
	mux := httphandler.Of(func(r *http.Request) http.HandlerFunc {
		return httphandler.MergeHeader(http.Header{
			"Test-Header": {"test header value"},
		},
			defresponse.Text(200, "Hello"),
		)
	})

	req := httptest.NewRequest("GET", "/", nil)
	res := httptest.NewRecorder()
	mux(res, req)

	fmt.Println(res.Header().Get("Test-Header"))
	// Output: test header value
}
