package e2ee

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
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
	// PublishE2EEDeviceAdded tells these users that a new device appeared and
	// needs adding to the chats they share with its owner (audit B-02).
	PublishE2EEDeviceAdded(userIDs []uuid.UUID, data any)
}

// ChatsOfUser lists the private chats a user takes part in, as
// (chatID, peerID) pairs. Injected in the composition root so this package
// does not import chats.
type ChatsOfUser func(ctx context.Context, userID uuid.UUID) ([]ChatPeer, error)

// ChatPeer is one private chat and the person on the other side of it.
type ChatPeer struct {
	ChatID uuid.UUID
	PeerID uuid.UUID
}

type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

type Service struct {
	pool       *pgxpool.Pool
	repo       Repository
	authz      Authorizer
	pub        Publisher
	peers      PeerCheck
	claimLimit ClaimLimit
	// chatsOf lists a user's private chats, to announce a new device to the
	// people it now has to talk to (audit B-02).
	chatsOf ChatsOfUser
	// joinLimit bounds how often one device may ask to be added to one chat.
	joinLimit JoinLimit
}

// JoinLimit reports whether this device may ask again, now, to be added to
// this chat.
type JoinLimit func(ctx context.Context, deviceID, chatID uuid.UUID) (bool, error)

// SetJoinLimit installs the bound on join requests. Without one the requests
// are not limited — the composition root always installs it.
func (s *Service) SetJoinLimit(l JoinLimit) { s.joinLimit = l }

// PeerCheck reports whether two users share a private chat.
type PeerCheck func(ctx context.Context, a, b uuid.UUID) (bool, error)

// ClaimLimit reports whether actor may claim another key package of target now.
type ClaimLimit func(ctx context.Context, actor, target uuid.UUID) (bool, error)

// SetClaimPolicy installs who may claim whose key packages (audit A-09).
func (s *Service) SetClaimPolicy(peers PeerCheck, limit ClaimLimit) {
	s.peers, s.claimLimit = peers, limit
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

	// A new device can read nothing in chats that already exist, and if it
	// builds its own group for one of them, neither can anyone else (audit
	// B-02). Tell both sides: whichever device is online adds it, and the
	// epoch gate makes sure exactly one of them succeeds.
	s.announceDevice(ctx, d)
	return d, nil
}

// announceDevice tells the owner's counterparts — and the owner's own other
// clients — that this device needs adding to their shared chats. Best-effort:
// a device that nobody adds now asks again on its next start, when it next
// uploads key packages, when it tries to send (RequestJoin), and any member
// opening the chat adds whatever devices its group is missing.
func (s *Service) announceDevice(ctx context.Context, d *Device) {
	if s.pub == nil || s.chatsOf == nil {
		return
	}
	chats, err := s.chatsOf(ctx, d.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "e2ee: list chats for a new device", "error", err)
		return
	}
	for _, c := range chats {
		s.publishDeviceAdded(c, d)
	}
}

func (s *Service) publishDeviceAdded(c ChatPeer, d *Device) {
	s.pub.PublishE2EEDeviceAdded([]uuid.UUID{c.PeerID, d.UserID}, map[string]any{
		"chatType": "private",
		"chatId":   c.ChatID,
		"deviceId": d.ID,
		"userId":   d.UserID,
	})
}

// RequestJoin asks the members of one private chat to add one of the actor's
// devices to its group. A device outside the group cannot send: the chat's
// first epoch is taken, so it may not build a group of its own, and only a
// member can let it in. Whoever of them is online does.
func (s *Service) RequestJoin(ctx context.Context, actor Actor, chatID, deviceID uuid.UUID) error {
	if err := s.authz.Private(ctx, chatID, actor.UserID); err != nil {
		return err
	}
	d, err := s.ownsActiveDevice(ctx, actor, deviceID)
	if err != nil {
		return err
	}
	if s.joinLimit != nil {
		allowed, err := s.joinLimit(ctx, deviceID, chatID)
		if err != nil {
			return err
		}
		if !allowed {
			return ErrRateLimited
		}
	}
	if s.pub == nil || s.chatsOf == nil {
		return nil
	}
	chats, err := s.chatsOf(ctx, actor.UserID)
	if err != nil {
		return err
	}
	for _, c := range chats {
		if c.ChatID == chatID {
			s.publishDeviceAdded(c, d)
			return nil
		}
	}
	return ErrNotFound
}

// SetChatsOfUser wires the lookup of a user's private chats, used to announce
// a new device to the people it now has to talk to.
func (s *Service) SetChatsOfUser(f ChatsOfUser) { s.chatsOf = f }

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
// checkWelcomeRecipients requires every recipient device to exist, be
// active, belong to the user the map names, and that user to be in the chat.
// Only private chats carry Welcomes today (group encryption is not built), so
// a group chat is refused rather than checked against a rule nobody uses.
func (s *Service) checkWelcomeRecipients(ctx context.Context, in PublishHandshakeInput) error {
	if in.ChatType != "private" {
		return ErrValidation
	}
	for deviceID, userID := range in.Recipients {
		d, err := s.repo.GetDevice(ctx, s.pool, deviceID)
		if err != nil {
			return ErrValidation
		}
		if d.RevokedAt != nil || d.UserID != userID {
			return ErrValidation
		}
		if err := s.authz.Private(ctx, in.ChatID, d.UserID); err != nil {
			return ErrValidation
		}
	}
	return nil
}

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
	d, err := s.ownsActiveDevice(ctx, actor, deviceID)
	if err != nil {
		return err
	}
	if err := s.repo.AddKeyPackages(ctx, s.pool, deviceID, packages); err != nil {
		return err
	}
	// A brand-new device registers first and uploads its packages second, so
	// the announcement made at registration reaches the other side while
	// there is nothing yet to add it with. Now there is.
	s.announceDevice(ctx, d)
	return nil
}

// ClaimKeyPackages consumes one key package per active device of userID —
// the caller is about to add that user to an MLS group. excludeDevice
// (uuid.Nil = none) lets a user claim their OWN other devices without
// burning the calling device's package; onlyDevice (uuid.Nil = all) claims
// for that one device alone, when it is the only one missing from a group.
func (s *Service) ClaimKeyPackages(ctx context.Context, actor Actor, userID, excludeDevice, onlyDevice uuid.UUID) ([]ClaimedKeyPackage, error) {
	// Claiming consumes the target's one-time packages, so it is not open to
	// anyone (audit A-09): your own other devices, or someone you already share
	// a private chat with — and a bounded number of times per pair. Without a
	// policy installed nothing but your own devices is claimable (fail closed).
	if userID != actor.UserID {
		if s.peers == nil {
			return nil, ErrNotFound
		}
		ok, err := s.peers(ctx, actor.UserID, userID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrNotFound
		}
		if s.claimLimit != nil {
			allowed, err := s.claimLimit(ctx, actor.UserID, userID)
			if err != nil {
				return nil, err
			}
			if !allowed {
				return nil, ErrRateLimited
			}
		}
	}
	claimed, err := s.repo.ClaimKeyPackages(ctx, s.pool, userID, excludeDevice, onlyDevice)
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
	if in.Kind == KindWelcome && (len(in.Recipients) == 0 || len(in.Recipients) > MaxWelcomeRecipients) {
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
		// Every recipient is checked before anything is stored. The map comes
		// from the client: its user ids decided who received a real-time
		// event, and nothing tied them to the devices or to the chat — any
		// user could be sent Welcome events for any chat (audit A-21).
		if err := s.checkWelcomeRecipients(ctx, in); err != nil {
			return err
		}
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

	// A commit moves the chat to a new epoch, and only one device may do that
	// at a time. Two of them committing at the same moment used to be accepted
	// both, and the group forked: two states, neither able to read the other
	// (audit B-02). The server cannot read the commit, but it can see that this
	// epoch is already taken and tell the loser to catch up.
	//
	// A commit without an epoch is from a client that predates this and is
	// accepted as before; the app has sent one since the feature existed.
	if in.Kind == KindCommit && in.Epoch != nil {
		moved, err := s.repo.AdvanceEpoch(ctx, s.pool, in.ChatType, in.ChatID, *in.Epoch)
		if err != nil {
			return err
		}
		if !moved {
			return ErrStaleEpoch
		}
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
