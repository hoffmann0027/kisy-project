import { useState } from "react";
import "./rating.css";
import { Rail } from "@widgets/rail/Rail";
import { Spinner } from "@shared/ui";
import { RatingAnalytics } from "@widgets/rating/RatingAnalytics";
import { RatingKanban } from "@widgets/rating/RatingKanban";
import { ProfileModal } from "@features/profile/ProfileModal";
import { NotificationsModal } from "@features/notifications/NotificationsModal";
import { FeedbackModal } from "@features/feedback/FeedbackModal";
import { useRatingAnalytics, useRatingBoard, useRatingMutations } from "@entities/rating/queries";

export function RatingPage() {
  const { data: board, isPending } = useRatingBoard();
  const { data: analytics } = useRatingAnalytics();
  const m = useRatingMutations();

  const [profile, setProfile] = useState(false);
  const [notifications, setNotifications] = useState(false);
  const [feedback, setFeedback] = useState(false);

  return (
    <div className="rating-shell">
      <Rail
        onProfile={() => setProfile(true)}
        onNotifications={() => setNotifications(true)}
        onFeedback={() => setFeedback(true)}
      />

      <main className="rating">
        <div className="rating__scroll">
          <h1 className="rating__heading">Рейтинг проектов</h1>

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
    </div>
  );
}
