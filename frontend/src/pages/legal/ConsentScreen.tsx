import { useState } from "react";
import { Button, Logo, Modal } from "@shared/ui";
import { LegalSections } from "./LegalPage";
import { LAST_UPDATED, PRIVACY_SECTIONS } from "./privacyContent";
import { RULES_LAST_UPDATED, RULES_SECTIONS } from "./rulesContent";
import "./legal.css";

// The screen nobody gets past without accepting the privacy policy and the
// community rules — two separate ticks, because they are two separate things
// to agree to, and Google Play's policy for user-generated content wants the
// rules accepted before anyone can publish.
//
// Each document opens right here, in a dialog: on the phone a link out would
// leave the app, and coming back would land on a fresh screen with both ticks
// gone.
//
// Shown in two places (app/router.tsx, app/guards.tsx): before sign-in and
// sign-up on a device where nobody has accepted yet, and after sign-in for an
// account that has not accepted the current versions.

interface Props {
  /** Called once both boxes are ticked and "Продолжить" pressed. */
  onAccept: () => void | Promise<void>;
  /** Wording for the account case: the texts may have changed since. */
  reason?: "first-run" | "account";
  /** Set while onAccept is in flight. */
  busy?: boolean;
}

type Doc = "privacy" | "rules" | null;

export function ConsentScreen({ onAccept, reason = "first-run", busy = false }: Props) {
  const [privacy, setPrivacy] = useState(false);
  const [rules, setRules] = useState(false);
  const [reading, setReading] = useState<Doc>(null);

  const both = privacy && rules;

  return (
    <div className="legal-screen">
      <article className="legal-card glass-surface consent">
        <header className="legal-head">
          <Logo size={56} className="legal-logo" />
          <h1 className="legal-title">Добро пожаловать в KISY</h1>
          <p className="legal-subtitle">
            {reason === "account"
              ? "Правила или политика обновились. Чтобы продолжить, примите их."
              : "Прежде чем войти или зарегистрироваться, примите два документа."}
          </p>
        </header>

        <label className="consent__row">
          <input
            type="checkbox"
            className="consent__box"
            checked={privacy}
            onChange={(e) => setPrivacy(e.target.checked)}
          />
          <span>
            Я прочитал(а) и принимаю{" "}
            <button type="button" className="consent__link" onClick={() => setReading("privacy")}>
              Политику конфиденциальности
            </button>
          </span>
        </label>

        <label className="consent__row">
          <input type="checkbox" className="consent__box" checked={rules} onChange={(e) => setRules(e.target.checked)} />
          <span>
            Я прочитал(а) и принимаю{" "}
            <button type="button" className="consent__link" onClick={() => setReading("rules")}>
              Правила сообщества
            </button>
          </span>
        </label>

        <Button block disabled={!both} loading={busy} onClick={() => void onAccept()}>
          Продолжить
        </Button>
        {!both && <p className="consent__hint">Отметьте оба пункта, чтобы продолжить.</p>}
      </article>

      <Modal
        open={reading === "privacy"}
        title={`Политика конфиденциальности · ${LAST_UPDATED}`}
        onClose={() => setReading(null)}
      >
        <div className="consent__doc">
          <LegalSections sections={PRIVACY_SECTIONS} />
        </div>
      </Modal>
      <Modal
        open={reading === "rules"}
        title={`Правила сообщества · ${RULES_LAST_UPDATED}`}
        onClose={() => setReading(null)}
      >
        <div className="consent__doc">
          <LegalSections sections={RULES_SECTIONS} />
        </div>
      </Modal>
    </div>
  );
}
