import { Link } from "react-router-dom";
import { Logo } from "@shared/ui";
import { t } from "@shared/i18n";
import { DELETION_STEPS, LAST_UPDATED, PRIVACY_SECTIONS, type LegalSection } from "./privacyContent";
import { RULES_LAST_UPDATED, RULES_SECTIONS } from "./rulesContent";
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

function LegalShell({ title, subtitle, sections }: { title: string; subtitle: string; sections: LegalSection[] }) {
  return (
    <div className="legal-screen">
      <article className="legal-card glass-surface">
        <header className="legal-head">
          <Logo size={56} className="legal-logo" />
          <h1 className="legal-title">{title}</h1>
          <p className="legal-subtitle">{subtitle}</p>
        </header>
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
  return (
    <LegalShell
      title={t("account.legal.privacyTitle")}
      subtitle={t("account.legal.updated", { date: LAST_UPDATED })}
      sections={PRIVACY_SECTIONS}
    />
  );
}

export function RulesPage() {
  return (
    <LegalShell
      title={t("account.legal.rulesTitle")}
      subtitle={t("account.legal.updated", { date: RULES_LAST_UPDATED })}
      sections={RULES_SECTIONS}
    />
  );
}

export function AccountDeletionPage() {
  return (
    <LegalShell
      title={t("account.legal.deletionTitle")}
      subtitle={t("account.legal.deletionSubtitle")}
      sections={DELETION_STEPS}
    />
  );
}
