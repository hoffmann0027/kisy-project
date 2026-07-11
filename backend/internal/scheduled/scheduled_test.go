package scheduled

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/messages"
)

func TestValidateContent(t *testing.T) {
	alg := int16(1)
	cases := []struct {
		name        string
		text        string
		ciphertext  []byte
		alg         *int16
		attachments int
		wantErr     bool
	}{
		{"text only", "hi", nil, nil, 0, false},
		{"ciphertext only", "", []byte{1}, &alg, 0, false},
		{"attachments only", "", nil, nil, 2, false},
		{"both bodies", "hi", []byte{1}, &alg, 0, true},
		{"nothing", "", nil, nil, 0, true},
		{"ciphertext without alg", "", []byte{1}, nil, 0, true},
		{"text too long", strings.Repeat("x", messages.MaxTextLength+1), nil, nil, 0, true},
		{"ciphertext too large", "", make([]byte, messages.MaxCiphertextBytes+1), &alg, 0, true},
	}
	for _, c := range cases {
		err := validateContent(c.text, c.ciphertext, c.alg, c.attachments)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: err=%v, wantErr=%v", c.name, err, c.wantErr)
		}
	}
}

func TestValidateSendAt(t *testing.T) {
	now := time.Now()
	if err := validateSendAt(now.Add(time.Minute), now); err != nil {
		t.Errorf("1min ahead should be valid: %v", err)
	}
	if err := validateSendAt(now.Add(-time.Minute), now); err != ErrValidation {
		t.Errorf("past should be invalid, got %v", err)
	}
	if err := validateSendAt(now.Add(time.Second), now); err != ErrValidation {
		t.Errorf("under MinDelay should be invalid, got %v", err)
	}
	if err := validateSendAt(now.Add(MaxDelay+time.Hour), now); err != ErrValidation {
		t.Errorf("beyond MaxDelay should be invalid, got %v", err)
	}
}

// Validation short-circuits before any repository or authorizer call.
func TestScheduleValidation(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil, nil)
	actor := Actor{UserID: uuid.New(), RoleLevel: 5}

	if _, err := svc.Schedule(t.Context(), Input{ChatType: "channel", ChatID: uuid.New(), Text: "hi", SendAt: time.Now().Add(time.Hour)}, actor); err != ErrValidation {
		t.Errorf("bad chat type: want ErrValidation, got %v", err)
	}
	if _, err := svc.Schedule(t.Context(), Input{ChatType: "private", ChatID: uuid.New(), SendAt: time.Now().Add(time.Hour)}, actor); err != ErrValidation {
		t.Errorf("empty content: want ErrValidation, got %v", err)
	}
	if _, err := svc.Schedule(t.Context(), Input{ChatType: "private", ChatID: uuid.New(), Text: "hi", SendAt: time.Now().Add(-time.Hour)}, actor); err != ErrValidation {
		t.Errorf("past sendAt: want ErrValidation, got %v", err)
	}
}
