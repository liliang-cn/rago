package web

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var staticFiles embed.FS

// GetStaticFiles returns the embedded static files
func GetStaticFiles() (fs.FS, error) {
	return fs.Sub(staticFiles, "dist")
}

// SetupStaticRoutes sets up static file serving routes
func SetupStaticRoutes(router *gin.Engine) error {
	fsys, err := GetStaticFiles()
	if err != nil {
		return err
	}

	// Serve index.html for root
	router.GET("/", func(c *gin.Context) {
		data, err := fs.ReadFile(fsys, "index.html")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read index.html"})
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	// Use NoRoute to handle SPA routing
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip API routes
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
			return
		}

		// Check if file exists in embedded FS
		if data, err := fs.ReadFile(fsys, path[1:]); err == nil {
			// Determine content type
			contentType := "text/plain"
			if containsFileExtension(path) {
				switch {
				case len(path) >= 3 && path[len(path)-3:] == ".js":
					contentType = "application/javascript"
				case len(path) >= 4 && path[len(path)-4:] == ".css":
					contentType = "text/css"
				case len(path) >= 4 && path[len(path)-4:] == ".png":
					contentType = "image/png"
				case len(path) >= 4 && path[len(path)-4:] == ".svg":
					contentType = "image/svg+xml"
				}
			}
			c.Data(http.StatusOK, contentType, data)
			return
		}

		// For SPA routes, serve index.html
		data, err := fs.ReadFile(fsys, "index.html")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read index.html"})
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	return nil
}

// Helper function to check if path contains a file extension
func containsFileExtension(path string) bool {
	extensions := []string{".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot"}
	for _, ext := range extensions {
		if len(path) >= len(ext) && path[len(path)-len(ext):] == ext {
			return true
		}
	}
	return false
}
