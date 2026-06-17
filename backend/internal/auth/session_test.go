package auth

import (
	"testing"
	"time"
)

func TestSession_CreateAndValidate(t *testing.T) {
	ss := NewSessionStore()
	token, err := ss.Create()
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if len(token) != 64 { // 32 bytes → 64 hex chars
		t.Errorf("token length = %d, want 64", len(token))
	}
	if !ss.IsValid(token) {
		t.Error("newly created token must be valid")
	}
}

func TestSession_TokensAreUnique(t *testing.T) {
	ss := NewSessionStore()
	t1, _ := ss.Create()
	t2, _ := ss.Create()
	if t1 == t2 {
		t.Error("two separate sessions must have different tokens")
	}
}

func TestSession_Delete(t *testing.T) {
	ss := NewSessionStore()
	token, _ := ss.Create()
	ss.Delete(token)
	if ss.IsValid(token) {
		t.Error("deleted token must not be valid")
	}
}

func TestSession_EmptyToken(t *testing.T) {
	ss := NewSessionStore()
	if ss.IsValid("") {
		t.Error("empty string must not be a valid token")
	}
}

func TestSession_UnknownToken(t *testing.T) {
	ss := NewSessionStore()
	if ss.IsValid("deadbeefdeadbeefdeadbeefdeadbeef") {
		t.Error("unknown token must not be valid")
	}
}

func TestSession_ExpiredToken(t *testing.T) {
	ss := NewSessionStore()
	token := "expired-token-for-testing"
	ss.mu.Lock()
	ss.sessions[token] = time.Now().Add(-time.Second) // already expired
	ss.mu.Unlock()
	if ss.IsValid(token) {
		t.Error("expired token must not be valid")
	}
}

func TestSession_DeleteNonexistent(t *testing.T) {
	ss := NewSessionStore()
	// Must not panic
	ss.Delete("nonexistent-token")
}

func TestSession_MultipleCreate(t *testing.T) {
	ss := NewSessionStore()
	tokens := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		tok, err := ss.Create()
		if err != nil {
			t.Fatalf("Create() error on iteration %d: %v", i, err)
		}
		if _, dup := tokens[tok]; dup {
			t.Fatalf("duplicate token generated at iteration %d", i)
		}
		tokens[tok] = struct{}{}
	}
}
