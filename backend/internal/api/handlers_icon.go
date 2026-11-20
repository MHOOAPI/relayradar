package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// handleAdminStationIcon accepts a small image upload and stores it as the
// station icon, served from /icons/.
func (s *Server) handleAdminStationIconReset(w http.ResponseWriter, r *http.Request) {
	if s.adminOnly(w, r) == nil {
		return
	}
	slug := r.PathValue("slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "missing slug")
		return
	}
	iconURL := "/api/v1/favicon/" + slug
	removeStationIconFiles(s.cfg.IconDir, slug)
	if err := s.svc.Repo().UpdateStationIcon(r.Context(), slug, iconURL); err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("station %s not found", slug))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"iconUrl": iconURL})
}

// handleAdminStationIcon accepts a small image upload and stores it as the
// station icon, served from /icons/.
func (s *Server) handleAdminStationIcon(w http.ResponseWriter, r *http.Request) {
	if s.adminOnly(w, r) == nil {
		return
	}
	slug := r.PathValue("slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "missing slug")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file")
		return
	}
	defer file.Close()
	buf, err := io.ReadAll(io.LimitReader(file, 2<<20))
	if err != nil || len(buf) == 0 {
		writeError(w, http.StatusBadRequest, "empty file")
		return
	}
	ext := ""
	switch http.DetectContentType(buf) {
	case "image/png":
		ext = ".png"
	case "image/jpeg":
		ext = ".jpg"
	case "image/webp":
		ext = ".webp"
	default:
		writeError(w, http.StatusBadRequest, "only png/jpeg/webp allowed")
		return
	}
	dir := s.cfg.IconDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	name := slug + ext
	if err := os.WriteFile(filepath.Join(dir, name), buf, 0o644); err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	for _, other := range []string{".png", ".jpg", ".webp"} {
		if other != ext {
			_ = os.Remove(filepath.Join(dir, slug+other))
		}
	}
	iconURL := "/icons/" + name
	if err := s.svc.Repo().UpdateStationIcon(r.Context(), slug, iconURL); err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("station %s not found", slug))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"iconUrl": iconURL})
}

func removeStationIconFiles(dir, slug string) {
	for _, ext := range []string{".png", ".jpg", ".webp"} {
		_ = os.Remove(filepath.Join(dir, slug+ext))
	}
}
