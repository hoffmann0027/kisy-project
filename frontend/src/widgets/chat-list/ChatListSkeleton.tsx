/**
 * Loading state of the chat list: four shimmering rows instead of a spinner
 * (design_handoff_kisy_mobile §3). Header, search and filters stay in place —
 * only the list itself is skeletonised.
 */
export function ChatListSkeleton({ rows = 4 }: { rows?: number }) {
  return (
    <div className="chatskel" aria-hidden="true">
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className="chatskel__row">
          <span className="chatskel__avatar" />
          <div className="chatskel__lines">
            <span className="chatskel__line chatskel__line--name" />
            <span className="chatskel__line chatskel__line--preview" />
          </div>
        </div>
      ))}
    </div>
  );
}
