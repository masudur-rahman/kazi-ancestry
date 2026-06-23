package auth

import "testing"

func TestSignVerifyRoundTrip(t *testing.T) {
	const secret = "test-secret"
	s := NewSession("a@b.com", "Aae", "admin")
	tok := Sign(s, secret)

	got, err := Verify(tok, secret)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.Email != s.Email || got.Role != "admin" {
		t.Errorf("round trip mismatch: %+v", got)
	}
}

func TestVerifyRejectsTamperAndWrongSecret(t *testing.T) {
	tok := Sign(NewSession("a@b.com", "A", "admin"), "secret-1")

	if _, err := Verify(tok, "secret-2"); err == nil {
		t.Error("expected failure with wrong secret")
	}
	if _, err := Verify(tok+"x", "secret-1"); err == nil {
		t.Error("expected failure with tampered signature")
	}
	if _, err := Verify("garbage", "secret-1"); err == nil {
		t.Error("expected failure with malformed token")
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	const secret = "s"
	s := Session{Email: "a@b.com", Role: "admin", Exp: 1} // long past
	if _, err := Verify(Sign(s, secret), secret); err == nil {
		t.Error("expected expired session to fail")
	}
}
