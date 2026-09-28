import { Link } from "react-router-dom";
import { Logo } from "@shared/ui";
import { DELETION_STEPS, LAST_UPDATED, PRIVACY_SECTIONS, type LegalSection } from "./privacyContent";
import "./legal.css";

// The two pages Google Play asks for by URL: the privacy policy, and a page
// that explains account deletion to someone who has not installed the app.
// Both are open without signing in — that is the point of them — and both are
// plain text so they stay readable when the app itself is down.

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
        {sections.map((section) => (
          <Section key={section.title} section={section} />
        ))}
        <footer className="legal-foot">
          <Link to="/login" className="auth-link">
            Вернуться в приложение
          </Link>
        </footer>
      </article>
    </div>
  );
}

export function PrivacyPage() {
  return (
    <LegalShell
      title="Политика конфиденциальности"
      subtitle={`KISY · обновлено ${LAST_UPDATED}`}
      sections={PRIVACY_SECTIONS}
    />
  );
}

export function AccountDeletionPage() {
  return (
    <LegalShell
      title="Удаление аккаунта"
      subtitle="KISY · как удалить аккаунт и что при этом происходит"
      sections={DELETION_STEPS}
    />
  );
}
