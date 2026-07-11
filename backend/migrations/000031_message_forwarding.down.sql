ALTER TABLE messages
    DROP COLUMN forwarded_from_message_id,
    DROP COLUMN forwarded_from_sender_id,
    DROP COLUMN forwarded_from_sender_name;
