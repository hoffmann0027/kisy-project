package e2ee

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// vouchContext must match VOUCH_CONTEXT in frontend/src/shared/crypto/identity.ts:
// a cross-signing vouch signs "KISY-device-vouch-v1" || newDevicePublicKey.
var vouchContext = []byte("KISY-device-vouch-v1")

// Authorizer decides whether an actor may access a chat; same injection
// pattern as messages.Authorizer (private → chats, group → groups).
type Authorizer struct {
	Private func(ctx context.Context, chatID, actorID uuid.UUID) error
	Group   func(ctx context.Context, groupID uuid.UUID, actorID uuid.UUID, actorLevel int) error
}

// Publisher fans handshake events out over websockets. Satisfied by the ws
// hub adapter; nil disables real-time delivery (clients still poll the
// mailbox endpoints).
type Publisher interface {
	// PublishE2EEHandshake tells a chat's connected members a commit/proposal
	// arrived so they advance their group state.
	PublishE2EEHandshake(chatType string, chatID uuid.UUID, data any)
	// PublishE2EEWelcome tells one user's connected clients a welcome awaits
	// one of their devices.
	PublishE2EEWelcome(userID uuid.UUID, data any)
}

type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

type Service struct {
	pool  *pgxpool.Pool
	repo  Repository
	authz Authorizer
	pub   Publisher
}

func NewService(pool *pgxpool.Pool, repo Repository, authz Authorizer) *Service {
	return &Service{pool: pool, repo: repo, authz: authz}
}

// SetPublisher wires the real-time publisher after construction (the hub
// is created later in the router assembly).
func (s *Service) SetPublisher(p Publisher) { s.pub = p }

func (s *Service) authorize(ctx context.Context, chatType string, chatID uuid.UUID, actor Actor) error {
	switch chatType {
	case "private":
		return s.authz.Private(ctx, chatID, actor.UserID)
	case "group":
		return s.authz.Group(ctx, chatID, actor.UserID, actor.RoleLevel)
	default:
		return ErrValidation
	}
}

// --- device directory ---

type RegisterDeviceInput struct {
	DeviceID   uuid.UUID
	Name       string
	Ed25519Pub []byte
	SignedBy   *uuid.UUID
	Signature  []byte
}

// RegisterDevice announces the actor's device public key. If a cross-signing
// vouch is supplied, the server sanity-checks it (an existing, non-revoked
// device of the same user must have signed this key). Clients still verify
// the chain themselves — the server check only keeps garbage out of the
// directory, it is not the trust root.
func (s *Service) RegisterDevice(ctx context.Context, actor Actor, in RegisterDeviceInput) (*Device, error) {
	if len(in.Ed25519Pub) != ed25519.PublicKeySize || in.DeviceID == uuid.Nil {
		return nil, ErrValidation
	}
	if len(in.Name) > MaxDeviceName {
		in.Name = in.Name[:MaxDeviceName]
	}

	if in.SignedBy != nil {
		signer, err := s.repo.GetDevice(ctx, s.pool, *in.SignedBy)
		if err != nil {
			return nil, ErrValidation
		}
		if signer.UserID != actor.UserID || signer.RevokedAt != nil {
			return nil, ErrForbidden
		}
		msg := append(append([]byte{}, vouchContext...), in.Ed25519Pub...)
		if !ed25519.Verify(signer.Ed25519Pub, msg, in.Signature) {
			return nil, ErrValidation
		}
	}

	d := &Device{
		ID:         in.DeviceID,
		UserID:     actor.UserID,
		Name:       in.Name,
		Ed25519Pub: in.Ed25519Pub,
		SignedBy:   in.SignedBy,
		Signature:  in.Signature,
	}
	if err := s.repo.UpsertDevice(ctx, s.pool, d); err != nil {
		return nil, err
	}
	return d, nil
}

// ListDevices returns a user's active devices. Any authenticated user may
// query the directory — public keys are public; trust comes from TOFU and
// safety-number verification on the client.
func (s *Service) ListDevices(ctx context.Context, userID uuid.UUID) ([]Device, error) {
	devices, err := s.repo.ListDevices(ctx, s.pool, userID)
	if err != nil {
		return nil, err
	}
	if devices == nil {
		devices = []Device{}
	}
	return devices, nil
}

func (s *Service) RevokeDevice(ctx context.Context, actor Actor, deviceID uuid.UUID) error {
	return s.repo.RevokeDevice(ctx, s.pool, deviceID, actor.UserID, time.Now())
}

// ownsActiveDevice guards uploads: the device must exist, belong to the
// actor and not be revoked.
func (s *Service) ownsActiveDevice(ctx context.Context, actor Actor, deviceID uuid.UUID) (*Device, error) {
	d, err := s.repo.GetDevice(ctx, s.pool, deviceID)
	if err != nil {
		return nil, err
	}
	if d.UserID != actor.UserID || d.RevokedAt != nil {
		return nil, ErrForbidden
	}
	return d, nil
}

// --- key packages ---

func (s *Service) UploadKeyPackages(ctx context.Context, actor Actor, deviceID uuid.UUID, packages [][]byte) error {
	if len(packages) == 0 || len(packages) > MaxBatchUpload {
		return ErrValidation
	}
	for _, kp := range packages {
		if len(kp) == 0 || len(kp) > MaxKeyPackageBytes {
			return ErrValidation
		}
	}
	if _, err := s.ownsActiveDevice(ctx, actor, deviceID); err != nil {
		return err
	}
	return s.repo.AddKeyPackages(ctx, s.pool, deviceID, packages)
}

// ClaimKeyPackages consumes one key package per active device of userID —
// the caller is about to add that user to an MLS group. excludeDevice
// (uuid.Nil = none) lets a user claim their OWN other devices without
// burning the calling device's package.
func (s *Service) ClaimKeyPackages(ctx context.Context, userID, excludeDevice uuid.UUID) ([]ClaimedKeyPackage, error) {
	claimed, err := s.repo.ClaimKeyPackages(ctx, s.pool, userID, excludeDevice)
	if err != nil {
		return nil, err
	}
	if claimed == nil {
		claimed = []ClaimedKeyPackage{}
	}
	return claimed, nil
}

func (s *Service) CountKeyPackages(ctx context.Context, actor Actor, deviceID uuid.UUID) (int, error) {
	if _, err := s.ownsActiveDevice(ctx, actor, deviceID); err != nil {
		return 0, err
	}
	return s.repo.CountKeyPackages(ctx, s.pool, deviceID)
}

// --- handshake mailbox ---

type PublishHandshakeInput struct {
	ChatType     string
	ChatID       uuid.UUID
	Kind         int16
	SenderDevice uuid.UUID
	Payload      []byte
	Epoch        *int64
	// Welcome targets: deviceID → owning userID (for WS notification).
	Recipients map[uuid.UUID]uuid.UUID
}

// PublishHandshake stores MLS handshake frames and fans them out. Commits and
// proposals go to the whole chat; welcomes go to specific devices.
func (s *Service) PublishHandshake(ctx context.Context, actor Actor, in PublishHandshakeInput) error {
	if len(in.Payload) == 0 || len(in.Payload) > MaxHandshakeBytes {
		return ErrValidation
	}
	if in.Kind != KindWelcome && in.Kind != KindCommit && in.Kind != KindProposal {
		return ErrValidation
	}
	if in.Kind == KindWelcome && len(in.Recipients) == 0 {
		return ErrValidation
	}
	if err := s.authorize(ctx, in.ChatType, in.ChatID, actor); err != nil {
		return err
	}
	if _, err := s.ownsActiveDevice(ctx, actor, in.SenderDevice); err != nil {
		return err
	}

	sender := in.SenderDevice
	if in.Kind == KindWelcome {
		for deviceID, userID := range in.Recipients {
			m := &GroupMessage{
				ChatType:        in.ChatType,
				ChatID:          in.ChatID,
				Kind:            in.Kind,
				SenderDevice:    &sender,
				RecipientDevice: &deviceID,
				Payload:         in.Payload,
				Epoch:           in.Epoch,
			}
			if err := s.repo.InsertGroupMessage(ctx, s.pool, m); err != nil {
				return err
			}
			if s.pub != nil {
				s.pub.PublishE2EEWelcome(userID, map[string]any{
					"chatType": in.ChatType,
					"chatId":   in.ChatID,
					"deviceId": deviceID,
				})
			}
		}
		return nil
	}

	m := &GroupMessage{
		ChatType:     in.ChatType,
		ChatID:       in.ChatID,
		Kind:         in.Kind,
		SenderDevice: &sender,
		Payload:      in.Payload,
		Epoch:        in.Epoch,
	}
	if err := s.repo.InsertGroupMessage(ctx, s.pool, m); err != nil {
		return err
	}
	if s.pub != nil {
		s.pub.PublishE2EEHandshake(in.ChatType, in.ChatID, map[string]any{
			"chatType": in.ChatType,
			"chatId":   in.ChatID,
			"id":       m.ID,
			"kind":     m.Kind,
			"epoch":    m.Epoch,
		})
	}
	return nil
}

func (s *Service) ListChatHandshake(ctx context.Context, actor Actor, chatType string, chatID, afterID uuid.UUID, limit int) ([]GroupMessage, error) {
	if err := s.authorize(ctx, chatType, chatID, actor); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	items, err := s.repo.ListChatHandshake(ctx, s.pool, chatType, chatID, afterID, limit)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []GroupMessage{}
	}
	return items, nil
}

func (s *Service) ListWelcomes(ctx context.Context, actor Actor, deviceID uuid.UUID) ([]GroupMessage, error) {
	if _, err := s.ownsActiveDevice(ctx, actor, deviceID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListWelcomes(ctx, s.pool, deviceID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []GroupMessage{}
	}
	return items, nil
}

func (s *Service) AckWelcome(ctx context.Context, actor Actor, deviceID, welcomeID uuid.UUID) error {
	if _, err := s.ownsActiveDevice(ctx, actor, deviceID); err != nil {
		return err
	}
	return s.repo.MarkWelcomeFetched(ctx, s.pool, welcomeID, deviceID, time.Now())
}

// --- encrypted backup ---

func (s *Service) PutBackup(ctx context.Context, actor Actor, blob []byte, kdfParams json.RawMessage) error {
	if len(blob) == 0 || len(blob) > MaxBackupBytes || len(kdfParams) == 0 {
		return ErrValidation
	}
	if !json.Valid(kdfParams) {
		return ErrValidation
	}
	return s.repo.PutBackup(ctx, s.pool, actor.UserID, blob, kdfParams)
}

func (s *Service) GetBackup(ctx context.Context, actor Actor) (*Backup, error) {
	return s.repo.GetBackup(ctx, s.pool, actor.UserID)
}

func (s *Service) DeleteBackup(ctx context.Context, actor Actor) error {
	return s.repo.DeleteBackup(ctx, s.pool, actor.UserID)
}
