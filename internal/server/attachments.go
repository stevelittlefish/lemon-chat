package server

import (
	"database/sql"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

func (s *Server) handleGetAttachment(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var att *store.Attachment
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
	if download {
		w.Header().Set("Content-Disposition", `attachment; filename="`+att.Filename+`"`)
	} else {
		w.Header().Set("Content-Disposition", `inline; filename="`+att.Filename+`"`)
	}
	w.Header().Set("Content-Type", att.MimeType)
	http.ServeFile(w, r, filepath.Join(s.cfg.Server.DataDir, att.DiskPath))
}
