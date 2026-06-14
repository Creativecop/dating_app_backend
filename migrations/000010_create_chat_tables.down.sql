ALTER TABLE conversation_participants
DROP CONSTRAINT IF EXISTS fk_conversation_participants_last_read_message;

ALTER TABLE conversations
DROP CONSTRAINT IF EXISTS fk_conversations_last_message;

DROP TRIGGER IF EXISTS trg_message_receipts_set_updated_at ON message_receipts;
DROP INDEX IF EXISTS idx_message_receipts_message;
DROP INDEX IF EXISTS idx_message_receipts_user;
DROP TABLE IF EXISTS message_receipts;

DROP TRIGGER IF EXISTS trg_messages_set_updated_at ON messages;
DROP INDEX IF EXISTS idx_messages_sender;
DROP INDEX IF EXISTS idx_messages_conversation_created;
DROP INDEX IF EXISTS idx_messages_conversation_id_id;
DROP INDEX IF EXISTS messages_client_message_unique;
DROP TABLE IF EXISTS messages;

DROP TRIGGER IF EXISTS trg_conversation_participants_set_updated_at ON conversation_participants;
DROP INDEX IF EXISTS idx_conversation_participants_conversation;
DROP INDEX IF EXISTS idx_conversation_participants_user;
DROP TABLE IF EXISTS conversation_participants;

DROP TRIGGER IF EXISTS trg_conversations_set_updated_at ON conversations;
DROP INDEX IF EXISTS idx_conversations_last_message_at;
DROP INDEX IF EXISTS idx_conversations_status;
DROP TABLE IF EXISTS conversations;
