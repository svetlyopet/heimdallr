package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var files embed.FS

func RegisterRoutes(router *gin.Engine) error {
	dist, err := fs.Sub(files, "dist")
	if err != nil {
		return err
	}

	indexHTML, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return err
	}

	fileServer := http.FileServer(http.FS(dist))

	router.NoRoute(func(ctx *gin.Context) {
		if strings.HasPrefix(ctx.Request.URL.Path, "/api") {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": http.StatusText(http.StatusNotFound),
			})
			return
		}

		requestPath := strings.TrimPrefix(path.Clean(ctx.Request.URL.Path), "/")

		if requestPath != "" && fileExists(dist, requestPath) {
			fileServer.ServeHTTP(ctx.Writer, ctx.Request)
			return
		}

		ctx.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})

	return nil
}

func fileExists(files fs.FS, name string) bool {
	file, err := files.Open(name)
	if err != nil {
		return false
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return false
	}

	return !stat.IsDir()
}
