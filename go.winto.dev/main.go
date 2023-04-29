package main

import (
	"fmt"
	"net/http"
)

func main() {
	panic(http.ListenAndServe(":8080", http.HandlerFunc(handler)))
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=600") // 10 minute

	if r.URL.Query().Get("go-get") != "1" {
		http.Redirect(w, r, "https://github.com/win-t/go.winto.dev", http.StatusFound)
		return
	}

	fmt.Fprintf(w,
		`<html><head><meta name="go-import" content="go.winto.dev git https://github.com/win-t/go.winto.dev"></head></html>`,
	)
}
