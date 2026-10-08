import { useState, type ReactNode } from "react";
import { Button, Logo, Modal } from "@shared/ui";
import { t } from "@shared/i18n";
import { LegalSections, TranslationNote } from "./LegalPage";
import { formatLegalDate, useLegalDocs } from "./legalContent";
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

/** A sentence with a link inside, kept whole for the translator: "{link}" marks where it goes. */
function WithLink({ text, link }: { text: string; link: ReactNode }) {
  const at = text.indexOf("{link}");
  if (at < 0) return <>{text}</>;
  return (
    <>
      {text.slice(0, at)}
      {link}
      {text.slice(at + "{link}".length)}
    </>
  );
}

export function ConsentScreen({ onAccept, reason = "first-run", busy = false }: Props) {
  const [privacy, setPrivacy] = useState(false);
  const [rules, setRules] = useState(false);
  const [reading, setReading] = useState<Doc>(null);
  const [original, setOriginal] = useState(false);
  const view = useLegalDocs(original);
  const docs = view.docs;
  const note =
    view.translated || original ? <TranslationNote original={original} onToggle={() => setOriginal((v) => !v)} /> : null;

  const both = privacy && rules;

  return (
    <div className="legal-screen">
      <article className="legal-card glass-surface consent">
        <header className="legal-head">
          <Logo size={56} className="legal-logo" />
          <h1 className="legal-title">{t("account.consent.title")}</h1>
          <p className="legal-subtitle">
            {reason === "account" ? t("account.consent.updated") : t("account.consent.firstRun")}
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
            <WithLink
              text={t("account.consent.acceptPrivacy")}
              link={
                <button type="button" className="consent__link" onClick={() => setReading("privacy")}>
                  {t("account.consent.privacyLink")}
                </button>
              }
            />
          </span>
        </label>

        <label className="consent__row">
          <input type="checkbox" className="consent__box" checked={rules} onChange={(e) => setRules(e.target.checked)} />
          <span>
            <WithLink
              text={t("account.consent.acceptRules")}
              link={
                <button type="button" className="consent__link" onClick={() => setReading("rules")}>
                  {t("account.consent.rulesLink")}
                </button>
              }
            />
          </span>
        </label>

        <Button block disabled={!both} loading={busy} onClick={() => void onAccept()}>
          {t("account.consent.continue")}
        </Button>
        {!both && <p className="consent__hint">{t("account.consent.hint")}</p>}
      </article>

      <Modal
        open={reading === "privacy"}
        title={`${t("account.legal.privacyTitle")} · ${formatLegalDate(docs.privacyUpdated)}`}
        onClose={() => setReading(null)}
      >
        <div className="consent__doc">
          {note}
          <LegalSections sections={docs.privacy} />
        </div>
      </Modal>
      <Modal
        open={reading === "rules"}
        title={`${t("account.legal.rulesTitle")} · ${formatLegalDate(docs.rulesUpdated)}`}
        onClose={() => setReading(null)}
      >
        <div className="consent__doc">
          {note}
          <LegalSections sections={docs.rules} />
        </div>
      </Modal>
    </div>
  );
}
