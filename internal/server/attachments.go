package server

import (
	"database/sql"
	"errors"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

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
