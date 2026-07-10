package e2ee

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes the E2EE directory/mailbox endpoints under /api/v1/e2ee.
// RequireAuth is applied by the router. Binary fields travel as base64 in
// JSON ([]byte's default Go encoding).
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

func (h *Handler) Routes(r chi.Router) {
	r.Post("/devices", h.registerDevice)
	r.Get("/users/{userID}/devices", h.listDevices)
	r.Delete("/devices/{deviceID}", h.revokeDevice)

	r.Post("/key-packages", h.uploadKeyPackages)
	r.Get("/key-packages/count", h.countKeyPackages)
	r.Post("/users/{userID}/key-packages/claim", h.claimKeyPackages)

	r.Post("/handshake", h.publishHandshake)
	r.Get("/handshake/{chatType}/{chatID}", h.listChatHandshake)
	r.Get("/welcomes", h.listWelcomes)
	r.Post("/welcomes/{welcomeID}/ack", h.ackWelcome)

	r.Put("/backup", h.putBackup)
	r.Get("/backup", h.getBackup)
	r.Delete("/backup", h.deleteBackup)
}

type registerDeviceRequest struct {
	DeviceID   string  `json:"deviceId"`
	Name       string  `json:"name"`
	Ed25519Pub string  `json:"ed25519Pub"` // base64
	SignedBy   *string `json:"signedBy"`
	Signature  string  `json:"signature"` // base64, required with signedBy
}

func (h *Handler) registerDevice(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	var req registerDeviceRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badRequest(w, r, "malformed JSON body")
		return
	}
	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		badRequest(w, r, "deviceId must be a valid UUID")
		return
	}
	pub, err := base64.StdEncoding.DecodeString(req.Ed25519Pub)
	if err != nil {
		badRequest(w, r, "ed25519Pub must be base64")
		return
	}
	in := RegisterDeviceInput{DeviceID: deviceID, Name: req.Name, Ed25519Pub: pub}
	if req.SignedBy != nil {
		signerID, err := uuid.Parse(*req.SignedBy)
		if err != nil {
			badRequest(w, r, "signedBy must be a valid UUID")
			return
		}
		sig, err := base64.StdEncoding.DecodeString(req.Signature)
		if err != nil || len(sig) == 0 {
			badRequest(w, r, "signature must be base64")
			return
		}
		in.SignedBy = &signerID
		in.Signature = sig
	}
	d, err := h.svc.RegisterDevice(r.Context(), actor, in)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"device": d})
}

func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.actor(r); !ok {
		unauth(w, r)
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		notFound(w, r)
		return
	}
	devices, err := h.svc.ListDevices(r.Context(), userID)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"devices": devices})
}

func (h *Handler) revokeDevice(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	deviceID, err := uuid.Parse(chi.URLParam(r, "deviceID"))
	if err != nil {
		notFound(w, r)
		return
	}
	if err := h.svc.RevokeDevice(r.Context(), actor, deviceID); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"revoked": true})
}

type uploadKeyPackagesRequest struct {
	DeviceID    string   `json:"deviceId"`
	KeyPackages []string `json:"keyPackages"` // base64 each
}

func (h *Handler) uploadKeyPackages(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	var req uploadKeyPackagesRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badRequest(w, r, "malformed JSON body")
		return
	}
	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		badRequest(w, r, "deviceId must be a valid UUID")
		return
	}
	packages := make([][]byte, 0, len(req.KeyPackages))
	for _, s := range req.KeyPackages {
		kp, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			badRequest(w, r, "keyPackages must be base64")
			return
		}
		packages = append(packages, kp)
	}
	if err := h.svc.UploadKeyPackages(r.Context(), actor, deviceID, packages); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"uploaded": len(packages)})
}

func (h *Handler) countKeyPackages(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	deviceID, err := uuid.Parse(r.URL.Query().Get("deviceId"))
	if err != nil {
		badRequest(w, r, "deviceId must be a valid UUID")
		return
	}
	n, err := h.svc.CountKeyPackages(r.Context(), actor, deviceID)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"available": n})
}

func (h *Handler) claimKeyPackages(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.actor(r); !ok {
		unauth(w, r)
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		notFound(w, r)
		return
	}
	claimed, err := h.svc.ClaimKeyPackages(r.Context(), userID)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"keyPackages": claimed})
}

type publishHandshakeRequest struct {
	ChatType     string `json:"chatType"`
	ChatID       string `json:"chatId"`
	Kind         string `json:"kind"` // "welcome" | "commit" | "proposal"
	SenderDevice string `json:"senderDevice"`
	Payload      string `json:"payload"` // base64
	Epoch        *int64 `json:"epoch"`
	// deviceId → userId of each welcome recipient.
	Recipients map[string]string `json:"recipients"`
}

func kindFromString(s string) (int16, bool) {
	switch s {
	case "welcome":
		return KindWelcome, true
	case "commit":
		return KindCommit, true
	case "proposal":
		return KindProposal, true
	default:
		return 0, false
	}
}

func (h *Handler) publishHandshake(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	var req publishHandshakeRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badRequest(w, r, "malformed JSON body")
		return
	}
	chatID, err := uuid.Parse(req.ChatID)
	if err != nil {
		badRequest(w, r, "chatId must be a valid UUID")
		return
	}
	senderDevice, err := uuid.Parse(req.SenderDevice)
	if err != nil {
		badRequest(w, r, "senderDevice must be a valid UUID")
		return
	}
	kind, ok := kindFromString(req.Kind)
	if !ok {
		badRequest(w, r, "kind must be welcome, commit or proposal")
		return
	}
	payload, err := base64.StdEncoding.DecodeString(req.Payload)
	if err != nil {
		badRequest(w, r, "payload must be base64")
		return
	}
	recipients := make(map[uuid.UUID]uuid.UUID, len(req.Recipients))
	for dev, usr := range req.Recipients {
		devID, err1 := uuid.Parse(dev)
		usrID, err2 := uuid.Parse(usr)
		if err1 != nil || err2 != nil {
			badRequest(w, r, "recipients must map device UUIDs to user UUIDs")
			return
		}
		recipients[devID] = usrID
	}
	err = h.svc.PublishHandshake(r.Context(), actor, PublishHandshakeInput{
		ChatType:     req.ChatType,
		ChatID:       chatID,
		Kind:         kind,
		SenderDevice: senderDevice,
		Payload:      payload,
		Epoch:        req.Epoch,
		Recipients:   recipients,
	})
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"published": true})
}

func (h *Handler) listChatHandshake(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	chatID, err := uuid.Parse(chi.URLParam(r, "chatID"))
	if err != nil {
		notFound(w, r)
		return
	}
	afterID := uuid.Nil
	if after := r.URL.Query().Get("afterId"); after != "" {
		afterID, err = uuid.Parse(after)
		if err != nil {
			badRequest(w, r, "afterId must be a valid UUID")
			return
		}
	}
	items, err := h.svc.ListChatHandshake(r.Context(), actor, chi.URLParam(r, "chatType"), chatID, afterID, 200)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"messages": items})
}

func (h *Handler) listWelcomes(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	deviceID, err := uuid.Parse(r.URL.Query().Get("deviceId"))
	if err != nil {
		badRequest(w, r, "deviceId must be a valid UUID")
		return
	}
	items, err := h.svc.ListWelcomes(r.Context(), actor, deviceID)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"welcomes": items})
}

func (h *Handler) ackWelcome(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	welcomeID, err := uuid.Parse(chi.URLParam(r, "welcomeID"))
	if err != nil {
		notFound(w, r)
		return
	}
	deviceID, err := uuid.Parse(r.URL.Query().Get("deviceId"))
	if err != nil {
		badRequest(w, r, "deviceId must be a valid UUID")
		return
	}
	if err := h.svc.AckWelcome(r.Context(), actor, deviceID, welcomeID); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"acked": true})
}

type putBackupRequest struct {
	Blob      string          `json:"blob"` // base64
	KDFParams json.RawMessage `json:"kdfParams"`
}

func (h *Handler) putBackup(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	var req putBackupRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badRequest(w, r, "malformed JSON body")
		return
	}
	blob, err := base64.StdEncoding.DecodeString(req.Blob)
	if err != nil {
		badRequest(w, r, "blob must be base64")
		return
	}
	if err := h.svc.PutBackup(r.Context(), actor, blob, req.KDFParams); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"saved": true})
}

func (h *Handler) getBackup(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	b, err := h.svc.GetBackup(r.Context(), actor)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"backup": b})
}

func (h *Handler) deleteBackup(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	if err := h.svc.DeleteBackup(r.Context(), actor); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"deleted": true})
}

// --- helpers ---

func unauth(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
}

func notFound(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "not found")
}

func badRequest(w http.ResponseWriter, r *http.Request, msg string) {
	httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, msg)
}

func fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		notFound(w, r)
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "not permitted")
	case errors.Is(err, ErrValidation):
		badRequest(w, r, "invalid request")
	default:
		// Chat authorizers surface their own not-found errors for hidden
		// chats; keep masking (404) instead of leaking existence via 500.
		notFound(w, r)
	}
}
