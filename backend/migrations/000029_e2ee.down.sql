DROP TABLE e2ee_backups;
DROP TABLE e2ee_membership_queue;
DROP TABLE e2ee_group_messages;
DROP TABLE e2ee_key_packages;
DROP TABLE e2ee_devices;

ALTER TABLE messages DROP CONSTRAINT messages_ciphertext_size;
ALTER TABLE messages
    DROP COLUMN ciphertext,
    DROP COLUMN alg,
    DROP COLUMN epoch,
    DROP COLUMN content_kind;
