CREATE TABLE IF NOT EXISTS mailboxes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS emails (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  mailbox_id UUID NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
  sender VARCHAR(255) NOT NULL,
  subject TEXT,
  body TEXT,
  state VARCHAR(50) DEFAULT 'received',
  flagged BOOLEAN DEFAULT false,
  read BOOLEAN DEFAULT false,
  received_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS folders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  mailbox_id UUID NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  UNIQUE(mailbox_id, name)
);

CREATE TABLE IF NOT EXISTS queue (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email_id UUID NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
  recipient VARCHAR(255) NOT NULL,
  state VARCHAR(50) DEFAULT 'pending'
);

CREATE INDEX idx_emails_mailbox_id ON emails(mailbox_id);
CREATE INDEX idx_emails_state ON emails(state);
CREATE INDEX idx_folders_mailbox_id ON folders(mailbox_id);
CREATE INDEX idx_queue_state ON queue(state);
