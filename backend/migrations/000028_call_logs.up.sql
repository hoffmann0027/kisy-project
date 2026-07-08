-- Voice call journal. Every 1:1 audio call (WebRTC, media P2P) is recorded here
-- regardless of outcome. status:
--   completed — answered and hung up normally (duration_seconds > 0),
--   missed    — rang out with no answer, or callee was busy/offline,
--   rejected  — callee actively declined,
--   canceled  — caller aborted before it was answered,
--   failed    — a media/negotiation error ended the call.
-- The row is created when the call is initiated and finalized on the terminal
-- signaling event; chat_id is the direct chat the two users share.
CREATE TABLE call_logs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    caller_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    callee_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_id          UUID NOT NULL,
    status           TEXT NOT NULL DEFAULT 'missed'
                         CHECK (status IN ('completed', 'missed', 'rejected', 'canceled', 'failed')),
    started_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    answered_at      TIMESTAMPTZ,
    ended_at         TIMESTAMPTZ,
    duration_seconds INT NOT NULL DEFAULT 0,

    CONSTRAINT call_logs_distinct_users CHECK (caller_id <> callee_id),
    CONSTRAINT call_logs_duration_nonneg CHECK (duration_seconds >= 0)
);

-- History is fetched per-user (either side) newest-first; also per-chat.
CREATE INDEX idx_call_logs_caller ON call_logs (caller_id, started_at DESC);
CREATE INDEX idx_call_logs_callee ON call_logs (callee_id, started_at DESC);
CREATE INDEX idx_call_logs_chat ON call_logs (chat_id, started_at DESC);
