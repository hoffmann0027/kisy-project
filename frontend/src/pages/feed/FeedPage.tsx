import { Icon } from "@shared/ui/icons";

// The feed of community posts.
//
// This is a placeholder, and deliberately a small honest one rather than a
// blank screen: the tab exists from this commit because an account without a
// rating board needs something in that slot, while the posts themselves arrive
// with communities in the next step. It offers nothing to press, because
// nothing here would work yet.

export function FeedPage() {
  return (
    <div
      style={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: 12,
        padding: "24px",
        paddingBottom: "calc(24px + var(--safe-bottom))",
        textAlign: "center",
      }}
    >
      <span style={{ color: "var(--color-text-tertiary)" }} aria-hidden="true">
        <Icon.Board size={40} />
      </span>
      <h1 style={{ margin: 0, fontSize: 19, fontWeight: 640 }}>Лента</h1>
      <p style={{ margin: 0, maxWidth: 320, color: "var(--color-text-secondary)", fontSize: 14 }}>
        Здесь появятся посты открытых сообществ. Раздел готовится.
      </p>
    </div>
  );
}
