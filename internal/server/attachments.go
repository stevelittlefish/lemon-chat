package server

import (
	"database/sql"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
)

func (s *Server) handleGetAttachment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	att, err := s.store.GetAttachment(id)
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
