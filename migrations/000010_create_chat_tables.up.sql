CREATE TABLE conversations (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  match_id BIGINT NOT NULL UNIQUE REFERENCES matches(id) ON DELETE CASCADE,

  status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',

  last_message_id BIGINT,
  last_message_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT conversations_status_check CHECK (
    status IN ('ACTIVE', 'READ_ONLY', 'CLOSED')
  )
);

CREATE INDEX idx_conversations_status
ON conversations(status);

CREATE INDEX idx_conversations_last_message_at
ON conversations(last_message_at DESC NULLS LAST, id DESC);

CREATE TRIGGER trg_conversations_set_updated_at
BEFORE UPDATE ON conversations
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE conversation_participants (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  last_read_message_id BIGINT,
  last_read_at TIMESTAMPTZ,

  muted_until TIMESTAMPTZ,
  hidden_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT conversation_participants_unique UNIQUE (
    conversation_id,
    user_id
  )
);

CREATE INDEX idx_conversation_participants_user
ON conversation_participants(user_id, hidden_at);

CREATE INDEX idx_conversation_participants_conversation
ON conversation_participants(conversation_id);

CREATE TRIGGER trg_conversation_participants_set_updated_at
BEFORE UPDATE ON conversation_participants
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE messages (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  match_id BIGINT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,

  sender_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  message_type VARCHAR(30) NOT NULL DEFAULT 'TEXT',
  body TEXT,

  client_message_id UUID NOT NULL,

  status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',

  deleted_at TIMESTAMPTZ,
  deleted_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT messages_type_check CHECK (
    message_type IN ('TEXT', 'SYSTEM')
  ),

  CONSTRAINT messages_status_check CHECK (
    status IN ('ACTIVE', 'DELETED')
  ),

  CONSTRAINT messages_body_check CHECK (
    message_type <> 'TEXT'
    OR status = 'DELETED'
    OR (body IS NOT NULL AND length(trim(body)) > 0)
  )
);

CREATE UNIQUE INDEX messages_client_message_unique
ON messages(sender_user_id, client_message_id);

CREATE INDEX idx_messages_conversation_id_id
ON messages(conversation_id, id DESC);

CREATE INDEX idx_messages_conversation_created
ON messages(conversation_id, created_at DESC, id DESC);

CREATE INDEX idx_messages_sender
ON messages(sender_user_id);

CREATE TRIGGER trg_messages_set_updated_at
BEFORE UPDATE ON messages
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE message_receipts (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  delivered_at TIMESTAMPTZ,
  seen_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT message_receipts_unique UNIQUE (
    message_id,
    user_id
  )
);

CREATE INDEX idx_message_receipts_user
ON message_receipts(user_id);

CREATE INDEX idx_message_receipts_message
ON message_receipts(message_id);

CREATE TRIGGER trg_message_receipts_set_updated_at
BEFORE UPDATE ON message_receipts
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE conversations
ADD CONSTRAINT fk_conversations_last_message
FOREIGN KEY (last_message_id)
REFERENCES messages(id)
ON DELETE SET NULL;

ALTER TABLE conversation_participants
ADD CONSTRAINT fk_conversation_participants_last_read_message
FOREIGN KEY (last_read_message_id)
REFERENCES messages(id)
ON DELETE SET NULL;

INSERT INTO conversations (match_id, status)
SELECT id, 'ACTIVE'
FROM matches
WHERE status = 'ACTIVE'
ON CONFLICT (match_id) DO NOTHING;

INSERT INTO conversation_participants (conversation_id, user_id)
SELECT c.id, m.user_low_id
FROM conversations c
JOIN matches m ON m.id = c.match_id
ON CONFLICT (conversation_id, user_id) DO NOTHING;

INSERT INTO conversation_participants (conversation_id, user_id)
SELECT c.id, m.user_high_id
FROM conversations c
JOIN matches m ON m.id = c.match_id
ON CONFLICT (conversation_id, user_id) DO NOTHING;
