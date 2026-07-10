// Package e2ee implements the server side of end-to-end encryption
// (docs/e2ee-design.md): a device/key-package directory (MLS Authentication
// Service role) and a handshake mailbox (MLS Delivery Service role).
//
// The server NEVER sees plaintext or private keys here — every payload is
// either a public key, a signed public key package, or protocol ciphertext.
// Clients verify signatures and apply TOFU; the directory is untrusted.
package e2ee

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("e2ee: not found")
	ErrForbidden  = errors.New("e2ee: forbidden")
	ErrValidation = errors.New("e2ee: validation failed")
)

// Handshake message kinds (e2ee_group_messages.kind).
const (
	KindWelcome  int16 = 1
	KindCommit   int16 = 2
	KindProposal int16 = 3
)

// Limits mirror the DB CHECK constraints so requests fail fast with a clean
// validation error instead of a constraint violation.
const (
	MaxKeyPackageBytes = 8192
	MaxHandshakeBytes  = 262144
	MaxBackupBytes     = 16 << 20
	MaxBatchUpload     = 100
	MaxDeviceName      = 128
)

type Device struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"userId"`
	Name       string     `json:"name"`
	Ed25519Pub []byte     `json:"ed25519Pub"` // JSON: base64 (Go's []byte default)
	SignedBy   *uuid.UUID `json:"signedBy"`
	Signature  []byte     `json:"signature,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	RevokedAt  *time.Time `json:"revokedAt"`
}

// ClaimedKeyPackage is one consumed one-time key package for one device.
type ClaimedKeyPackage struct {
	DeviceID   uuid.UUID `json:"deviceId"`
	KeyPackage []byte    `json:"keyPackage"`
}

type GroupMessage struct {
	ID              uuid.UUID  `json:"id"`
	ChatType        string     `json:"chatType"`
	ChatID          uuid.UUID  `json:"chatId"`
	Kind            int16      `json:"kind"`
	SenderDevice    *uuid.UUID `json:"senderDevice"`
	RecipientDevice *uuid.UUID `json:"recipientDevice,omitempty"`
	Payload         []byte     `json:"payload"`
	Epoch           *int64     `json:"epoch"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type Backup struct {
	Blob      []byte    `json:"blob"`
	KDFParams []byte    `json:"kdfParams"` // opaque JSON: salt/opslimit/memlimit
	UpdatedAt time.Time `json:"updatedAt"`
}
