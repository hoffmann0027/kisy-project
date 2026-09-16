package moderation

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// What the people who run a group are told. The reason is always in it: a
// sanction its recipients cannot understand is one they cannot act on.

func subject(g *groupRow) (nominative, dative string) {
	if g.Kind == "community" {
		return "Сообщество", "Сообществу"
	}
	return "Группа", "Группе"
}

func (s *Service) announce(ctx context.Context, g *groupRow, kind string, sanction Sanction, activeWarns int) {
	if s.notifier == nil {
		return
	}
	nom, dat := subject(g)
	var text, url string
	switch kind {
	case KindWarn:
		text = fmt.Sprintf("%s «%s» вынесено предупреждение (%d из %d): %s", dat, g.Name, activeWarns, WarnLimit, sanction.Reason)
		url = "/group/" + g.ID.String()
	case KindMute:
		until := "бессрочно"
		if sanction.ExpiresAt != nil {
			until = "до " + sanction.ExpiresAt.UTC().Format("02.01.2006 15:04") + " (UTC)"
		}
		text = fmt.Sprintf("%s «%s» замучено %s — посты не показываются в ленте: %s", nom, g.Name, until, sanction.Reason)
		url = "/group/" + g.ID.String()
	case KindDelete:
		text = fmt.Sprintf("%s «%s» удалено: %s", nom, g.Name, sanction.Reason)
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
		Payload: map[string]any{
			"action":      kind,
			"groupId":     g.ID,
			"groupName":   g.Name,
			"groupKind":   g.Kind,
			"reason":      sanction.Reason,
			"activeWarns": activeWarns,
			"warnLimit":   WarnLimit,
			"expiresAt":   sanction.ExpiresAt,
			"text":        text,
		},
	})
}

// announceRestore tells the founder their group is back.
func (s *Service) announceRestore(ctx context.Context, g *groupRow) {
	if s.notifier == nil {
		return
	}
	nom, _ := subject(g)
	text := fmt.Sprintf("%s «%s» восстановлено", nom, g.Name)
	s.deliver(ctx, Notice{
		Recipients: []uuid.UUID{g.CreatedBy},
		Text:       text,
		URL:        "/group/" + g.ID.String(),
		Payload: map[string]any{
			"action": "restore", "groupId": g.ID, "groupName": g.Name, "groupKind": g.Kind, "text": text,
		},
	})
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
