-- The current MLS epoch of every end-to-end encrypted chat (audit B-02).
--
-- Why the server, which cannot read a single message, tracks this: two devices
-- of the same chat can commit at the same moment, and until now the delivery
-- service accepted both. The group then forked — two states, each valid to its
-- own author, and nothing either of them sends is readable by the other.
--
-- The server does not need to understand a commit to stop that. It only needs
-- to know which epoch the chat is in and refuse anything that is not moving it
-- forward. One of the two racing commits wins; the other is told to catch up.
CREATE TABLE e2ee_group_epochs (
    chat_type  VARCHAR(16) NOT NULL CHECK (chat_type IN ('private', 'group')),
    chat_id    UUID NOT NULL,
    epoch      BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (chat_type, chat_id)
);

-- Chats that already exist start wherever their members are: the first commit
-- after this migration sets the epoch, and everything before it is accepted as
-- it always was.
