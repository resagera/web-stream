package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var files embed.FS

func Handler() http.Handler {
	static, err := fs.Sub(files, "static")
	if err != nil {
		panic(err)
	}
	fileserver := http.FileServer(http.FS(static))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin" {
			http.Redirect(w, r, "/admin.html", http.StatusTemporaryRedirect)
			return
		}
		fileserver.ServeHTTP(w, r)
	})
}
