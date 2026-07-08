import { useState } from "react";
import "./rating.css";
import { Rail } from "@widgets/rail/Rail";
import { Spinner } from "@shared/ui";
import { RatingAnalytics } from "@widgets/rating/RatingAnalytics";
import { RatingKanban } from "@widgets/rating/RatingKanban";
import { ProfileModal } from "@features/profile/ProfileModal";
import { NotificationsModal } from "@features/notifications/NotificationsModal";
import { FeedbackModal } from "@features/feedback/FeedbackModal";
import { NotesModal } from "@features/notes/NotesModal";
import { useRatingAnalytics, useRatingBoard, useRatingMutations } from "@entities/rating/queries";

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

export function RatingPage() {
  const { data: board, isPending } = useRatingBoard();
  const { data: analytics } = useRatingAnalytics();
  const m = useRatingMutations();

  const [profile, setProfile] = useState(false);
  const [notifications, setNotifications] = useState(false);
  const [feedback, setFeedback] = useState(false);
  const [notes, setNotes] = useState(false);

  return (
    <div className="rating-shell">
      <Rail
        onProfile={() => setProfile(true)}
        onNotifications={() => setNotifications(true)}
        onFeedback={() => setFeedback(true)}
        onNotes={() => setNotes(true)}
      />

      <main className="rating">
        <div className="rating__scroll">
          <div className="rating__topbar">
            <h1 className="rating__heading">Рейтинг проектов</h1>
            <a className="rating-add rating__export" href={`${API_BASE}/rating/export.csv`}>
              Экспорт CSV
            </a>
          </div>

          {analytics && <RatingAnalytics data={analytics} />}

          {isPending || !board ? (
            <div style={{ display: "flex", justifyContent: "center", padding: 40 }}>
              <Spinner size={28} />
            </div>
          ) : (
            <RatingKanban board={board} m={m} />
          )}
        </div>
      </main>

      <ProfileModal open={profile} onClose={() => setProfile(false)} />
      <NotificationsModal open={notifications} onClose={() => setNotifications(false)} />
      <FeedbackModal open={feedback} onClose={() => setFeedback(false)} />
      <NotesModal open={notes} onClose={() => setNotes(false)} />
    </div>
  );
}
