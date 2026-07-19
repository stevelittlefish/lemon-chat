package tasks

import (
	"log"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

const (
	conversationCleanupInterval = time.Minute
	sessionCleanupInterval      = time.Hour
)

func StartCleanupWorker(st *store.Store, onDeleted func()) {
	go func() {
		deleteStaleConversations(st, onDeleted)
		deleteExpiredSessions(st)

		conversationTicker := time.NewTicker(conversationCleanupInterval)
		sessionTicker := time.NewTicker(sessionCleanupInterval)
		defer conversationTicker.Stop()
		defer sessionTicker.Stop()

		for {
			select {
			case <-conversationTicker.C:
				deleteStaleConversations(st, onDeleted)
			case <-sessionTicker.C:
				deleteExpiredSessions(st)
			}
		}
	}()
}

func deleteStaleConversations(st *store.Store, onDeleted func()) {
	convs, msgs, err := st.DeleteStaleConversations()
	if err != nil {
		log.Printf("cleanup worker: %v", err)
		return
	}
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
