package server

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/gif"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const avatarSize = 256

func (s *Server) avatarDir() string {
	return filepath.Join(filepath.Dir(s.cfg.Server.DBPath), "avatars")
}

// mimeFromFilename returns the MIME type for a filename. receiveAvatar always
// produces ".jpg" files, so only the JPEG case is reachable in practice.
func mimeFromFilename(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return "application/octet-stream"
	}
	ext := strings.ToLower(parts[len(parts)-1])
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	}
	log.Printf("avatars: unhandled extension %q, falling back to octet-stream", ext)
	return "application/octet-stream"
}

// receiveAvatar parses a multipart upload (field "avatar"), validates it is an
// image ≤5 MB, then crops and resizes it to a 256×256 JPEG.
// Pass ?crop=top to crop from the top of the image instead of the center.
// Always returns the ".jpg" extension so callers can ignore the original format.
func receiveAvatar(r *http.Request) ([]byte, string, error) {
	cropTop := r.URL.Query().Get("crop") == "top"
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

	// Sniff the first 512 bytes to validate it's an image before reading all.
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
	raw := append(buf[:n], rest...)

	processed, err := processAvatar(raw, cropTop)
	if err != nil {
		return nil, "", fmt.Errorf("could not process image: %w", err)
	}
	return processed, ".jpg", nil
}

// processAvatar decodes raw image bytes, crops to a square, scales to
// avatarSize×avatarSize, and returns a JPEG-encoded result.
// When cropTop is true the square is taken from the top of the image rather
// than the center — better for full-body portrait images.
func processAvatar(data []byte, cropTop bool) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	var srcRect image.Rectangle
	if cropTop {
		// Largest square anchored at the top of the image.
		topH := h / 2
		size := w
		if topH < size {
			size = topH
		}
		x0 := b.Min.X + (w-size)/2
		srcRect = image.Rect(x0, b.Min.Y, x0+size, b.Min.Y+size)
	} else {
		// Center-crop to largest square.
		size := w
		if h < size {
			size = h
		}
		x0 := b.Min.X + (w-size)/2
		y0 := b.Min.Y + (h-size)/2
		srcRect = image.Rect(x0, y0, x0+size, y0+size)
	}

	// Scale to target size using CatmullRom (high quality, similar to Lanczos).
	dst := image.NewRGBA(image.Rect(0, 0, avatarSize, avatarSize))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, srcRect, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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

	dest := filepath.Join(dir, filename)
	log.Printf("avatars: saving %s (%d bytes)", dest, len(data))
	if err := os.WriteFile(dest, data, 0644); err != nil {
		return "", err
	}
	log.Printf("avatars: saved %s", dest)
	return filename, nil
}

func (s *Server) deleteAvatarFile(filename string) {
	if filename == "" {
		return
	}
	os.Remove(filepath.Join(s.avatarDir(), filename))
}

func (s *Server) serveAvatarFile(w http.ResponseWriter, r *http.Request, filename string) {
	path := filepath.Join(s.avatarDir(), filename)
	f, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		internalError(w, err)
		return
	}
	w.Header().Set("Content-Type", mimeFromFilename(filename))
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, filename, info.ModTime(), f)
}
