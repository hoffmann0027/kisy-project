# KISY Enterprise Technical Specification - Part 4 (Database)

## Database Platform

PostgreSQL 16. UTF-8. UUID primary keys. Timestamps in UTC. Soft delete
where appropriate. ACID transactions.

## Table: users

Accounts, profile, avatar, status, role_id, password_hash, timestamps.

## Table: roles

Level 1-10 with permission sets.

## Table: permissions

Fine-grained permission catalog.

## Table: role_permissions

Role-to-permission mapping.

## Table: groups

Group metadata and minimum clearance.

## Table: group_members

Membership, join dates, mute settings.

## Table: private_chats

One-to-one conversations.

## Table: messages

Text, sender, chat, timestamps, deleted flag.

## Table: attachments

Files linked to messages.

## Table: reactions

Emoji reactions.

## Table: sessions

Refresh tokens, devices, IP hash, expiry.

## Table: invitation_tokens

Creator, expiry, usage state.

## Table: audit_logs

Every privileged action.

## Table: notifications

Unread notifications.

## Table: favorites

Pinned/favorite chats.

## Table: search_index

Optimized search metadata.

## Indexes

Create indexes for usernames, chat lookups, timestamps, unread counts,
invitations, audit queries, full-text search.

## Constraints

Foreign keys, unique usernames, cascading rules only where safe, check
constraints for role levels and token validity.

## Backups

Nightly encrypted backups, point-in-time recovery, backup verification
jobs, retention policy.

## Performance

Partition audit and message tables if growth requires. Use connection
pooling and prepared statements.

## Database Design Note 1

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.

## Database Design Note 2

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.

## Database Design Note 3

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.

## Database Design Note 4

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.

## Database Design Note 5

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.

## Database Design Note 6

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.

## Database Design Note 7

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.

## Database Design Note 8

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.

## Database Design Note 9

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.

## Database Design Note 10

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.

## Database Design Note 11

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.

## Database Design Note 12

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.

## Database Design Note 13

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.

## Database Design Note 14

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.

## Database Design Note 15

Specify normalization, indexing strategy, migration rules, rollback
strategy, data retention, consistency guarantees, query optimization and
monitoring.
