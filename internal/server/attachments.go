package server

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

// maxUploadBytes caps a single user-uploaded attachment.
const maxUploadBytes = 10 << 20 // 10 MB

// uploadImageExt maps an accepted image content type to a file extension.
var uploadImageExt = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// handleUploadAttachment receives a user-uploaded image (multipart field
// "file"), stores it under the conversation, and returns the new attachment.
// The attachment is created unlinked; it is bound to a message when the user
// sends the message it was attached to. Image-only for now (vision input).
func (s *Server) handleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	convID, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := s.store.GetConversation(convID, user.ID); notFoundOr500(w, err) {
		return
	}

	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()
	if header.Size > maxUploadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file too large (max 10 MB)")
		return
	}

	// Sniff the first 512 bytes to validate the content type before reading all.
	head := make([]byte, 512)
	n, _ := file.Read(head)
	ct := http.DetectContentType(head[:n])
	ext, allowed := uploadImageExt[ct]
	if !allowed {
		writeError(w, http.StatusUnsupportedMediaType, "file must be a PNG, JPEG, GIF, or WebP image")
		return
	}
	rest, err := io.ReadAll(io.LimitReader(file, maxUploadBytes))
	if err != nil {
		internalError(w, err)
		return
	}
	data := append(head[:n], rest...)

	relDir := filepath.Join("attachments", randomID())
	absDir := filepath.Join(s.cfg.Server.DataDir, relDir)
	if err := os.MkdirAll(absDir, 0755); err != nil {
		internalError(w, fmt.Errorf("could not create storage directory: %w", err))
		return
	}
	filename := "upload" + ext
	if err := os.WriteFile(filepath.Join(absDir, filename), data, 0644); err != nil {
		internalError(w, fmt.Errorf("could not write upload to disk: %w", err))
		return
	}
	diskPath := filepath.Join(relDir, filename)

	att, err := s.store.CreateUploadAttachment(convID, filename, ct, diskPath)
	if err != nil {
		internalError(w, err)
		return
	}
	log.Printf("Uploading attachment id=%d conversation_id=%d user_id=%d mime=%s bytes=%d", att.ID, convID, user.ID, ct, len(data))
	writeJSON(w, http.StatusCreated, att)
}

// imageDataURL reads an image attachment from disk and returns a base64 data
// URL suitable for a chat-completions image_url content part.
func (s *Server) imageDataURL(att *store.Attachment) (string, error) {
	data, err := os.ReadFile(filepath.Join(s.cfg.Server.DataDir, att.DiskPath))
	if err != nil {
		return "", err
	}
	return "data:" + att.MimeType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// uploadImagePartsByMessage returns, for a conversation, the image_url content
// parts grouped by the user message they are attached to. Only ready image
// uploads bound to a message are included; unreadable files are skipped with a
// log line rather than failing the whole request.
func (s *Server) uploadImagePartsByMessage(convID int64) (map[int64][]any, error) {
	atts, err := s.store.ListAttachmentsByConversation(convID)
	if err != nil {
		return nil, err
	}
	out := map[int64][]any{}
	for i := range atts {
		a := atts[i]
		if a.Source != "upload" || a.MessageID == nil || a.Status != "ready" || !strings.HasPrefix(a.MimeType, "image/") {
			continue
		}
		url, err := s.imageDataURL(&a)
		if err != nil {
			log.Printf("uploadImagePartsByMessage: skipping attachment id=%d: %v", a.ID, err)
			continue
		}
		out[*a.MessageID] = append(out[*a.MessageID], imagePart(url))
	}
	return out, nil
}

func (s *Server) handleGetAttachment(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var att *store.Attachment
	var err error
	if user.IsAdmin {
		att, err = s.store.GetAttachment(id)
	} else {
		att, err = s.store.GetAttachmentForUser(id, user.ID)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not found")
		} else {
			internalError(w, err)
		}
		return
	}

	download := r.URL.Query().Get("download") == "1"
	disposition := "inline"
	if download {
		disposition = "attachment"
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": att.Filename}))
	w.Header().Set("Content-Type", att.MimeType)
	http.ServeFile(w, r, filepath.Join(s.cfg.Server.DataDir, att.DiskPath))
}
