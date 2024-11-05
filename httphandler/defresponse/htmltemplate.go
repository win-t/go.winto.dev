package defresponse

import (
	"html/template"
	"net/http"
)

func HTMLTemplate(status int, t *template.Template, name string, data any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)

		if name == "" {
			t.Execute(w, data)
		} else {
			t.ExecuteTemplate(w, name, data)
		}
	}
}
