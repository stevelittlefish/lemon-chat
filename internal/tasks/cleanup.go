package tasks

import (
	"log"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

const cleanupInterval = time.Minute

func StartCleanupWorker(st *store.Store, onDeleted func()) {
	go func() {
		deleteStaleConversations(st, onDeleted)
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			deleteStaleConversations(st, onDeleted)
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
