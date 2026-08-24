package push

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes Web Push subscription endpoints.
type Handler struct {
	svc    *Service
	userID func(*http.Request) (uuid.UUID, bool)
}

func NewHandler(svc *Service, userID func(*http.Request) (uuid.UUID, bool)) *Handler {
	return &Handler{svc: svc, userID: userID}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/vapid-public-key", h.publicKey)
	r.Post("/subscribe", h.subscribe)
	r.Post("/unsubscribe", h.unsubscribe)
	r.Post("/device", h.registerDevice)
	r.Post("/device/unregister", h.unregisterDevice)
}

func (h *Handler) publicKey(w http.ResponseWriter, r *http.Request) {
	httpresponse.OK(w, r, http.StatusOK, map[string]any{
		"publicKey": h.svc.PublicKey(),
		"enabled":   h.svc.Enabled(),
		// Lets the mobile app skip asking for the notification permission when
		// the server could not deliver anything anyway.
		"mobileEnabled": h.svc.MobileEnabled(),
	})
}

type subscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (h *Handler) subscribe(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.userID(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	var req subscribeRequest
	if err := httpjson.Decode(w, r, &req); err != nil || req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid subscription")
		return
	}
	if err := h.svc.Subscribe(r.Context(), uid, Subscription{Endpoint: req.Endpoint, P256dh: req.Keys.P256dh, Auth: req.Keys.Auth}); err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to subscribe")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"subscribed": true})
}

type unsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

func (h *Handler) unsubscribe(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.userID(r); !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	var req unsubscribeRequest
	if err := httpjson.Decode(w, r, &req); err != nil || req.Endpoint == "" {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid request")
		return
	}
	_ = h.svc.Unsubscribe(r.Context(), req.Endpoint)
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"unsubscribed": true})
}

type deviceRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

// maxDeviceTokenLen bounds what is stored: real FCM tokens are ~160-200 chars.
const maxDeviceTokenLen = 1024

func (h *Handler) registerDevice(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.userID(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	var req deviceRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid device registration")
		return
	}
	if req.Platform == "" {
		req.Platform = "android"
	}
	if req.Token == "" || len(req.Token) > maxDeviceTokenLen || (req.Platform != "android" && req.Platform != "ios") {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid device registration")
		return
	}
	if err := h.svc.RegisterDevice(r.Context(), uid, Device{Token: req.Token, Platform: req.Platform}); err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to register device")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"registered": true})
}

func (h *Handler) unregisterDevice(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.userID(r); !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	var req deviceRequest
	if err := httpjson.Decode(w, r, &req); err != nil || req.Token == "" || len(req.Token) > maxDeviceTokenLen {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid request")
		return
	}
	_ = h.svc.UnregisterDevice(r.Context(), req.Token)
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"unregistered": true})
}
