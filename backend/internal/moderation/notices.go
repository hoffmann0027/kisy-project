package moderation

import (
	"context"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/i18n"
)

// What the people who run a group are told. The reason is always in it: a
// sanction its recipients cannot understand is one they cannot act on.

// kindKey picks the wording for the group's kind: Russian agrees the verb
// with the noun («Сообщество удалено», «Группа удалена»), so the two are
// separate messages rather than one with the noun swapped in.
func kindKey(g *groupRow, action string) string {
	if g.Kind == "community" {
		return "moderation." + action + ".community"
	}
	return "moderation." + action + ".group"
}

func (s *Service) announce(ctx context.Context, g *groupRow, kind string, sanction Sanction, activeWarns int) {
	if s.notifier == nil {
		return
	}
	var text i18n.Msg
	var url string
	switch kind {
	case KindWarn:
		text = i18n.M(kindKey(g, "warn"), g.Name, activeWarns, WarnLimit, sanction.Reason)
		url = "/group/" + g.ID.String()
	case KindMute:
		text = i18n.M(kindKey(g, "mute"), g.Name, muteUntil{sanction.ExpiresAt}, sanction.Reason)
		url = "/group/" + g.ID.String()
	case KindDelete:
		text = i18n.M(kindKey(g, "delete"), g.Name, sanction.Reason)
		url = "/"
	default:
		return
	}
	recipients, err := s.repo.recipients(ctx, s.pool, g.ID)
	if err != nil {
		s.log.Warn("moderation: recipients", "error", err)
		return
	}
	s.deliver(ctx, Notice{
		Recipients: recipients,
		Text:       text,
		URL:        url,
		// The app words the notice itself from these fields, in its own
		// language; "text" is for apps from before that.
		Payload: map[string]any{
			"action":      kind,
			"groupId":     g.ID,
			"groupName":   g.Name,
			"groupKind":   g.Kind,
			"reason":      sanction.Reason,
			"activeWarns": activeWarns,
			"warnLimit":   WarnLimit,
			"expiresAt":   sanction.ExpiresAt,
			"text":        text.In(i18n.Default),
		},
	})
}

// announceRestore tells the founder their group is back.
func (s *Service) announceRestore(ctx context.Context, g *groupRow) {
	if s.notifier == nil {
		return
	}
	text := i18n.M(kindKey(g, "restored"), g.Name)
	s.deliver(ctx, Notice{
		Recipients: []uuid.UUID{g.CreatedBy},
		Text:       text,
		URL:        "/group/" + g.ID.String(),
		Payload: map[string]any{
			"action": "restore", "groupId": g.ID, "groupName": g.Name, "groupKind": g.Kind, "text": text.In(i18n.Default),
		},
	})
}

// muteUntil is how long a mute lasts, in the reader's words.
type muteUntil struct{ at *time.Time }

func (m muteUntil) In(l i18n.Lang) string {
	if m.at == nil {
		return i18n.T(l, "moderation.untilForever")
	}
	layout := "2006-01-02 15:04"
	if l == "ru" || l == "uk" {
		layout = "02.01.2006 15:04"
	}
	return i18n.T(l, "moderation.until", m.at.UTC().Format(layout))
}

func (s *Service) deliver(ctx context.Context, n Notice) {
	if err := s.notifier.Notify(ctx, n); err != nil {
		// The sanction is already in force and audited; a lost notification
		// must not undo it, only be visible in the logs.
		s.log.Warn("moderation: notify", "error", err)
	}
}

func (s *Service) touchFeed(ctx context.Context) {
	if s.feedChanged != nil {
		s.feedChanged(ctx)
	}
}

func (s *Service) touchGroup(groupID uuid.UUID) {
	if s.groupChanged != nil {
		s.groupChanged(groupID)
	}
}
