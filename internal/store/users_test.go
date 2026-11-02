package store

import (
	"errors"
	"testing"
)

func TestDeleteUserRemovesEverything(t *testing.T) {
	s := newTestStore(t)
	victim, conv := newTestConversation(t, s)
	other, err := s.CreateUser("other", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.CreateSession(victim.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.SetUserAvatar(victim.ID, "user-1.png"); err != nil {
		t.Fatal(err)
	}
	up, err := s.CreateUploadAttachment(conv.ID, "a.png", "image/png", "attachments/abc/a.png")
	if err != nil {
		t.Fatal(err)
	}
	msg, err := s.CreateMessage(conv.ID, "user", "hi", nil, nil, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.LinkAttachmentsToMessage(msg.ID, conv.ID, []int64{up.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateCompletion(victim.ID, "m"); err != nil {
		t.Fatal(err)
	}

	// A public character the victim made, which another user is chatting with.
	char, err := s.CreateCharacter("Lemon", "char-model", nil, nil, nil, victim.ID, "public", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetCharacterAvatar(char.ID, "char-1.png"); err != nil {
		t.Fatal(err)
	}
	otherConv, err := s.CreateConversation(other.ID, nil, nil, &char.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateMessage(otherConv.ID, "assistant", "hello", &char.ID, nil, nil, nil, ""); err != nil {
		t.Fatal(err)
	}

	paths, avatars, err := s.DeleteUser(victim.ID)
	if err != nil {
		t.Fatalf("delete user: %v", err)
	}
	if len(paths) != 1 || paths[0] != "attachments/abc/a.png" {
		t.Fatalf("paths = %v", paths)
	}
	if len(avatars) != 2 {
		t.Fatalf("avatars = %v, want user and character avatars", avatars)
	}

	if _, err := s.UserByID(victim.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("user still exists: %v", err)
	}
	for _, table := range []string{"session", "conversation", "completion"} {
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE user_id = ?`, victim.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("%d %s row(s) left", n, table)
		}
	}

	got, err := s.GetConversation(otherConv.ID, other.ID)
	if err != nil {
		t.Fatalf("other user's conversation: %v", err)
	}
	if got.CharacterID != nil || got.Model == nil || *got.Model != "char-model" {
		t.Fatalf("other conversation character=%v model=%v, want reverted to char-model", got.CharacterID, got.Model)
	}

	if _, _, err := s.DeleteUser(victim.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete: got %v, want ErrNotFound", err)
	}
}
