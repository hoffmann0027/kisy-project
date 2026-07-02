package password

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	const plain = "correct horse battery staple 42"

	encoded, err := Hash(plain)
	if err != nil {
		t.Fatalf("Hash() error: %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$") {
		t.Fatalf("unexpected encoding prefix: %s", encoded)
	}

	ok, err := Verify(plain, encoded)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if !ok {
		t.Fatal("Verify() = false for correct password")
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	encoded, err := Hash("the-real-password-123")
	if err != nil {
		t.Fatalf("Hash() error: %v", err)
	}

	ok, err := Verify("not-the-password-456", encoded)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if ok {
		t.Fatal("Verify() = true for wrong password")
	}
}

func TestHashUniqueSalts(t *testing.T) {
	a, err := Hash("same-password-every-time")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Hash("same-password-every-time")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("two hashes of the same password are identical; salt is not random")
	}
}

func TestVerifyMalformed(t *testing.T) {
	cases := []string{
		"",
		"plaintext",
		"$argon2id$v=19$m=65536,t=3,p=4$onlyonepart",
		"$bcrypt$v=19$m=65536,t=3,p=4$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=65536,t=3,p=4$!!!$aGFzaA",
	}
	for _, c := range cases {
		if _, err := Verify("x", c); err == nil {
			t.Errorf("Verify(%q) expected error, got nil", c)
		}
	}
}
