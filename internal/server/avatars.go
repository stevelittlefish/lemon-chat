package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) avatarDir() string {
	return filepath.Join(filepath.Dir(s.cfg.Server.DBPath), "avatars")
}

func mimeFromFilename(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return "application/octet-stream"
	}
	switch strings.ToLower(parts[len(parts)-1]) {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	}
	return "application/octet-stream"
}

func extFromContentType(ct string) string {
	switch {
	case strings.Contains(ct, "jpeg"):
		return ".jpg"
	case strings.Contains(ct, "png"):
		return ".png"
	case strings.Contains(ct, "gif"):
		return ".gif"
	case strings.Contains(ct, "webp"):
		return ".webp"
	}
	return ".jpg"
}

// receiveAvatar parses a multipart upload, validates it is an image ≤5 MB,
// and returns the file bytes and detected extension.
func receiveAvatar(r *http.Request) ([]byte, string, error) {
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		return nil, "", fmt.Errorf("file too large or invalid form")
	}
	file, header, err := r.FormFile("avatar")
	if err != nil {
		return nil, "", fmt.Errorf("missing avatar field")
	}
	defer file.Close()

	if header.Size > 5<<20 {
		return nil, "", fmt.Errorf("file too large (max 5 MB)")
	}

	// Read the first 512 bytes to detect content type, then read the rest.
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	ct := http.DetectContentType(buf[:n])
	if !strings.HasPrefix(ct, "image/") {
		return nil, "", fmt.Errorf("file must be an image")
	}

	// Collect all bytes (sniffed prefix + remainder).
	rest, err := io.ReadAll(file)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file")
	}
	data := append(buf[:n], rest...)

	// Prefer extension from original filename, fall back to detected type.
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" || !isImageExt(ext) {
		ext = extFromContentType(ct)
	}
	return data, ext, nil
}

func isImageExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	}
	return false
}

// writeAvatar saves avatar bytes to the avatar directory with the given prefix
// (e.g. "user-1"), removing any pre-existing file for that prefix first.
func (s *Server) writeAvatar(prefix string, data []byte, ext string) (string, error) {
	dir := s.avatarDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	filename := prefix + ext

	// Remove old avatar for this entity (may have a different extension).
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix+".") && e.Name() != filename {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}

	if err := os.WriteFile(filepath.Join(dir, filename), data, 0644); err != nil {
		return "", err
	}
	return filename, nil
}

func (s *Server) deleteAvatarFile(filename string) {
	if filename == "" {
		return
	}
	os.Remove(filepath.Join(s.avatarDir(), filename))
}

func (s *Server) serveAvatarFile(w http.ResponseWriter, filename string) {
	path := filepath.Join(s.avatarDir(), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	w.Header().Set("Content-Type", mimeFromFilename(filename))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
