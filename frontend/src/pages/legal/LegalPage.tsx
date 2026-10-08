import { useState } from "react";
import { Link } from "react-router-dom";
import { Logo } from "@shared/ui";
import { t } from "@shared/i18n";
import { formatLegalDate, useLegalDocs, type LegalDocs, type LegalSection } from "./legalContent";
import "./legal.css";

// The pages Google Play asks for by URL — the privacy policy, and a page that
// explains account deletion to someone who has not installed the app — plus
// the community rules every account accepts. All are open without signing in
// (that is the point of them), and all are plain text so they stay readable
// when the app itself is down.

/** The body of a legal document; also rendered inside the consent screen. */
export function LegalSections({ sections }: { sections: LegalSection[] }) {
  return (
    <>
      {sections.map((section) => (
        <Section key={section.title} section={section} />
      ))}
    </>
  );
}

function Section({ section }: { section: LegalSection }) {
  return (
    <section className="legal-section">
      <h2 className="legal-section__title">{section.title}</h2>
      {section.body.map((paragraph) =>
        paragraph.startsWith("— ") || /^\d\./.test(paragraph) ? (
          <p key={paragraph} className="legal-item">
            {paragraph}
          </p>
        ) : (
          <p key={paragraph} className="legal-text">
            {paragraph}
          </p>
        ),
      )}
    </section>
  );
}

/**
 * Over a translated document: it is a translation, the Russian text is the one
 * that binds, and here is that text.
 */
export function TranslationNote({ original, onToggle }: { original: boolean; onToggle: () => void }) {
  return (
    <p className="legal-note">
      {!original && t("account.legal.translationNote")}{" "}
      <button type="button" className="consent__link" onClick={onToggle}>
        {original ? t("account.legal.showTranslation") : t("account.legal.showOriginal")}
      </button>
    </p>
  );
}

/** A document in the language on screen, with a way to the binding original. */
function useDocument(): { docs: LegalDocs; note: React.ReactNode } {
  const [original, setOriginal] = useState(false);
  const view = useLegalDocs(original);
  // The note stays while the original is shown, so the way back does too.
  const offersTranslation = view.translated || original;
  return {
    docs: view.docs,
    note: offersTranslation ? <TranslationNote original={original} onToggle={() => setOriginal((v) => !v)} /> : null,
  };
}

function LegalShell({
  title,
  subtitle,
  note,
  sections,
}: {
  title: string;
  subtitle: string;
  note: React.ReactNode;
  sections: LegalSection[];
}) {
  return (
    <div className="legal-screen">
      <article className="legal-card glass-surface">
        <header className="legal-head">
          <Logo size={56} className="legal-logo" />
          <h1 className="legal-title">{title}</h1>
          <p className="legal-subtitle">{subtitle}</p>
        </header>
        {note}
        <LegalSections sections={sections} />
        <footer className="legal-foot">
          <Link to="/login" className="auth-link">
            {t("account.legal.backToApp")}
          </Link>
        </footer>
      </article>
    </div>
  );
}

export function PrivacyPage() {
  const { docs, note } = useDocument();
  return (
    <LegalShell
      title={t("account.legal.privacyTitle")}
      subtitle={t("account.legal.updated", { date: formatLegalDate(docs.privacyUpdated) })}
      note={note}
      sections={docs.privacy}
    />
  );
}

export function RulesPage() {
  const { docs, note } = useDocument();
  return (
    <LegalShell
      title={t("account.legal.rulesTitle")}
      subtitle={t("account.legal.updated", { date: formatLegalDate(docs.rulesUpdated) })}
      note={note}
      sections={docs.rules}
    />
  );
}

export function AccountDeletionPage() {
  const { docs, note } = useDocument();
  return (
    <LegalShell
      title={t("account.legal.deletionTitle")}
      subtitle={t("account.legal.deletionSubtitle")}
      note={note}
      sections={docs.deletion}
    />
  );
}
