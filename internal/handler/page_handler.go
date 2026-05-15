package handler

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// SPAHandler serves a Vue/React SPA from an fs.FS.
// Static assets are served directly; all other routes fall back to index.html.
type SPAHandler struct {
	fs        http.FileSystem
	indexHTML []byte
}

func NewSPAHandler(root fs.FS) *SPAHandler {
	idx, _ := fs.ReadFile(root, "index.html")
	return &SPAHandler{fs: http.FS(root), indexHTML: idx}
}

func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/")
	if p == "" {
		p = "index.html"
	}

	// Try to serve the file directly (assets, favicon, etc.)
	if f, err := h.fs.Open(p); err == nil {
		stat, _ := f.Stat()
		if !stat.IsDir() {
			ext := path.Ext(p)
			switch ext {
			case ".css", ".js", ".woff2", ".woff", ".ttf":
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			case ".png", ".jpg", ".jpeg", ".svg", ".ico", ".gif", ".webp":
				w.Header().Set("Cache-Control", "public, max-age=604800")
			}
			http.ServeContent(w, r, stat.Name(), stat.ModTime(), f.(readSeeker))
			f.Close()
			return
		}
		f.Close()
	}

	// SPA fallback: serve index.html for all other routes
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(h.indexHTML)
}

type readSeeker interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
}
