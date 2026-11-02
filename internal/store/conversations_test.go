package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestConversation(t *testing.T, s *Store) (*User, *Conversation) {
	t.Helper()
	user, err := s.CreateUser("conv-test-user", nil, false, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	model := "m"
	conv, err := s.CreateConversation(user.ID, nil, &model, nil)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	return user, conv
}

func backdate(t *testing.T, s *Store, table string, id int64, age time.Duration) {
	t.Helper()
	ts := time.Now().UTC().Add(-age).Format(time.RFC3339)
	if _, err := s.db.Exec(`UPDATE `+table+` SET created_at = ? WHERE id = ?`, ts, id); err != nil {
		t.Fatalf("backdate %s %d: %v", table, id, err)
	}
}

func TestDeleteStaleConversationsWithUnsentUpload(t *testing.T) {
	s := newTestStore(t)
	_, conv := newTestConversation(t, s)
	if _, err := s.CreateUploadAttachment(conv.ID, "a.png", "image/png", "attachments/abc/a.png"); err != nil {
		t.Fatalf("create upload: %v", err)
	}
	backdate(t, s, "conversation", conv.ID, time.Hour)

	convs, _, paths, err := s.DeleteStaleConversations()
	if err != nil {
		t.Fatalf("delete stale conversations: %v", err)
	}
	if convs != 1 {
		t.Fatalf("deleted %d conversations, want 1", convs)
	}
	if len(paths) != 1 || paths[0] != "attachments/abc/a.png" {
		t.Fatalf("paths = %v, want [attachments/abc/a.png]", paths)
	}
}

func TestDeleteUnlinkedUploads(t *testing.T) {
	s := newTestStore(t)
	_, conv := newTestConversation(t, s)
	old, err := s.CreateUploadAttachment(conv.ID, "old.png", "image/png", "attachments/old/old.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateUploadAttachment(conv.ID, "new.png", "image/png", "attachments/new/new.png"); err != nil {
		t.Fatal(err)
	}
	linked, err := s.CreateUploadAttachment(conv.ID, "sent.png", "image/png", "attachments/sent/sent.png")
	if err != nil {
		t.Fatal(err)
	}
	msg, err := s.CreateMessage(conv.ID, "user", "hi", nil, nil, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.LinkAttachmentsToMessage(msg.ID, conv.ID, []int64{linked.ID}); err != nil {
		t.Fatal(err)
	}
	backdate(t, s, "attachment", old.ID, 48*time.Hour)
	backdate(t, s, "attachment", linked.ID, 48*time.Hour)

	deleted, paths, err := s.DeleteUnlinkedUploads(24 * time.Hour)
	if err != nil {
		t.Fatalf("delete unlinked uploads: %v", err)
	}
	if deleted != 1 || len(paths) != 1 || paths[0] != "attachments/old/old.png" {
		t.Fatalf("deleted=%d paths=%v, want 1 [attachments/old/old.png]", deleted, paths)
	}
}

func TestForkConversationCopiesUploads(t *testing.T) {
	s := newTestStore(t)
	user, conv := newTestConversation(t, s)
	up, err := s.CreateUploadAttachment(conv.ID, "a.png", "image/png", "attachments/abc/a.png")
	if err != nil {
		t.Fatal(err)
	}
	userMsg, err := s.CreateMessage(conv.ID, "user", "what is this", nil, nil, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.LinkAttachmentsToMessage(userMsg.ID, conv.ID, []int64{up.ID}); err != nil {
		t.Fatal(err)
	}
	reply, err := s.CreateMessage(conv.ID, "assistant", "a lemon", nil, nil, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	fork, err := s.ForkConversation(conv.ID, user.ID, reply.ID)
	if err != nil {
		t.Fatalf("fork: %v", err)
	}
	atts, err := s.ListAttachmentsByConversation(fork.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(atts) != 1 {
		t.Fatalf("fork has %d attachments, want 1", len(atts))
	}
	a := atts[0]
	if a.Source != "upload" || a.MessageID == nil {
		t.Fatalf("fork attachment source=%q message_id=%v, want upload bound to a message", a.Source, a.MessageID)
	}
	msgs, err := s.ListMessages(fork.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || msgs[0].Role != "user" || *a.MessageID != msgs[0].ID {
		t.Fatalf("fork upload bound to message %d, want fork's first user message", *a.MessageID)
	}

	// Deleting the source must leave the shared file for the fork.
	paths, err := s.DeleteConversation(conv.ID, user.ID)
	if err != nil {
		t.Fatalf("delete source: %v", err)
	}
	if len(paths) != 0 {
		t.Fatalf("delete source returned shared paths %v", paths)
	}
}

func TestDeleteConversationWithFailedAttachment(t *testing.T) {
	s := newTestStore(t)
	user, conv := newTestConversation(t, s)
	att, err := s.CreatePendingAttachment("call_1", conv.ID, "img", "img.png", "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FailAttachment(att.ID, "boom"); err != nil {
		t.Fatal(err)
	}
	paths, err := s.DeleteConversation(conv.ID, user.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	// The failed attachment has an empty disk_path; removing its files must
	// not touch anything outside attachments/.
	dataDir := t.TempDir()
	keep := filepath.Join(dataDir, "lemon.db")
	if err := os.WriteFile(keep, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	RemoveAttachmentFiles(dataDir, paths)
	RemoveAttachmentFiles(dataDir, []string{"", ".", "attachments", "../x/y", "attachments/../lemon.db"})
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("data dir file removed: %v", err)
	}

	attDir := filepath.Join(dataDir, "attachments", "abc")
	if err := os.MkdirAll(attDir, 0o700); err != nil {
		t.Fatal(err)
	}
	RemoveAttachmentFiles(dataDir, []string{"attachments/abc/a.png"})
	if _, err := os.Stat(attDir); !os.IsNotExist(err) {
		t.Fatalf("attachment dir not removed: %v", err)
	}
}
