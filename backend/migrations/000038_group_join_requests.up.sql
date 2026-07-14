-- Join requests for request-to-join ('request') groups. A cleared user who is
-- not yet a member applies; an editor/owner/CEO approves or rejects.
CREATE TABLE group_join_requests (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id     UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       VARCHAR(16) NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'approved', 'rejected')),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_by   UUID REFERENCES users(id),
    decided_at   TIMESTAMPTZ
);

-- At most one active (pending) request per user per group.
CREATE UNIQUE INDEX uq_group_join_requests_pending
    ON group_join_requests (group_id, user_id)
    WHERE status = 'pending';

-- Approvers list a group's pending requests.
CREATE INDEX idx_group_join_requests_group_status
    ON group_join_requests (group_id, status);
