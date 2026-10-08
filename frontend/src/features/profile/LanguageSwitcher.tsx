import { currentLang, LANG_NAMES, LANGS, setLanguage, t, type Lang } from "@shared/i18n";

// The language of the interface. Changing it restarts the screen in the new
// one. The label also says "Language" in English: someone who landed in a
// language they cannot read still has to be able to find the way out.
export function LanguageSwitcher() {
  const lang = currentLang();
  return (
    <label style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
      <span style={{ fontSize: 14 }}>
        {t("account.language")}
        {lang !== "en" && " · Language"}
      </span>
      <select className="ui-input" style={{ width: "auto" }} value={lang} onChange={(e) => setLanguage(e.target.value as Lang)}>
        {LANGS.map((l) => (
          <option key={l} value={l}>
            {LANG_NAMES[l]}
          </option>
        ))}
      </select>
    </label>
  );
}
