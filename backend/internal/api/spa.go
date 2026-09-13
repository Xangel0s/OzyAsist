package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var embeddedFrontend embed.FS

// RegisterSPA configura el enrutador para servir la SPA de React de forma embebida
func RegisterSPA(r *gin.Engine) {
	distFS, err := fs.Sub(embeddedFrontend, "dist")
	if err != nil {
		return
	}

	fileServer := http.FileServer(http.FS(distFS))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/ws") || strings.HasPrefix(path, "/health") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint no encontrado"})
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		// Si el archivo existe físicamente en el build embebido, servirlo
		f, err := distFS.Open(cleanPath)
		if err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// Fallback para rutas cliente de React Router (/chats, /proyectos, etc.)
		indexFile, err := distFS.Open("index.html")
		if err == nil {
			_ = indexFile.Close()
			c.Request.URL.Path = "/"
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "Frontend embebido no disponible"})
	})
}
