package tasks

import (
	"log"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

const (
	conversationCleanupInterval = time.Minute
	sessionCleanupInterval      = time.Hour
	unlinkedUploadMaxAge        = 24 * time.Hour
)

func StartCleanupWorker(st *store.Store, dataDir string, onDeleted func()) {
	go func() {
		deleteStaleConversations(st, dataDir, onDeleted)
		deleteExpiredSessions(st)
		deleteUnlinkedUploads(st, dataDir)

		conversationTicker := time.NewTicker(conversationCleanupInterval)
		sessionTicker := time.NewTicker(sessionCleanupInterval)
		defer conversationTicker.Stop()
		defer sessionTicker.Stop()

		for {
			select {
			case <-conversationTicker.C:
				deleteStaleConversations(st, dataDir, onDeleted)
			case <-sessionTicker.C:
				deleteExpiredSessions(st)
				deleteUnlinkedUploads(st, dataDir)
			}
		}
	}()
}

func deleteStaleConversations(st *store.Store, dataDir string, onDeleted func()) {
	convs, msgs, paths, err := st.DeleteStaleConversations()
	if err != nil {
		log.Printf("cleanup worker: %v", err)
		return
	}
	store.RemoveAttachmentFiles(dataDir, paths)
	if convs > 0 {
		log.Printf("cleanup worker: deleted %d stale conversation(s) and %d message(s)", convs, msgs)
		onDeleted()
	}
}

func deleteExpiredSessions(st *store.Store) {
	deleted, err := st.DeleteExpiredSessions()
	if err != nil {
		log.Printf("cleanup worker: deleting expired sessions: %v", err)
		return
	}
	if deleted > 0 {
		log.Printf("cleanup worker: deleted %d expired session(s)", deleted)
	}
}

func deleteUnlinkedUploads(st *store.Store, dataDir string) {
	deleted, paths, err := st.DeleteUnlinkedUploads(unlinkedUploadMaxAge)
	if err != nil {
		log.Printf("cleanup worker: deleting unlinked uploads: %v", err)
		return
	}
	store.RemoveAttachmentFiles(dataDir, paths)
	if deleted > 0 {
		log.Printf("cleanup worker: deleted %d unsent upload(s)", deleted)
	}
}
