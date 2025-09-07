package web

import (
	"embed"
	"io/fs"
	"net/http"
)

// Embed the entire dist directory
//
//go:embed dist/*
var content embed.FS

// GetFileSystem returns the embedded filesystem for serving
func GetFileSystem() (http.FileSystem, error) {
	fsys, err := fs.Sub(content, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(fsys), nil
}

// Handler returns an http.Handler that serves the embedded frontend
func Handler() http.Handler {
	fsys, err := GetFileSystem()
	if err != nil {
		panic(err)
	}
	return http.FileServer(fsys)
}