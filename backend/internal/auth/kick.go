package auth

import "github.com/google/uuid"

// SessionKicker ends the live connections (WebSockets) riding on sessions
// that were just revoked. Revoking the row stops new requests; without this a
// socket opened earlier keeps receiving and sending for as long as it stays
// up (audit A-03). Satisfied by *ws.Hub.
type SessionKicker interface {
	// KickSession ends the sockets of one session.
	KickSession(userID, sessionID uuid.UUID)
	// KickUser ends every socket of the user except those of keep
	// (uuid.Nil keeps none).
	KickUser(userID, keep uuid.UUID)
}

type noKick struct{}

func (noKick) KickSession(uuid.UUID, uuid.UUID) {}
func (noKick) KickUser(uuid.UUID, uuid.UUID)    {}

// SetSessionKicker wires the connection kick. Without one, sockets still end
// on their own session re-check.
func (s *Service) SetSessionKicker(k SessionKicker) {
	if k == nil {
		k = noKick{}
	}
	s.kick = k
}
