-- Core identity and access control
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  full_name VARCHAR(255),
  role VARCHAR(50) DEFAULT 'user',
  mfa_enabled BOOLEAN DEFAULT false,
  mfa_secret VARCHAR(255),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS teams (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  description TEXT,
  owner_id UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS team_members (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role VARCHAR(50) DEFAULT 'member',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(team_id, user_id)
);

-- Infrastructure management
CREATE TABLE IF NOT EXISTS nodes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  status VARCHAR(50) DEFAULT 'online',
  cpu_cores INT,
  memory_gb INT,
  disk_gb INT,
  ip_address INET,
  region VARCHAR(100),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS deployments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  status VARCHAR(50) DEFAULT 'pending',
  version VARCHAR(100),
  deployed_by UUID REFERENCES users(id),
  deployed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Monitoring and alerts
CREATE TABLE IF NOT EXISTS alerts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  title VARCHAR(255) NOT NULL,
  description TEXT,
  severity VARCHAR(50) DEFAULT 'info',
  status VARCHAR(50) DEFAULT 'firing',
  source VARCHAR(100),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  resolved_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  actor_id UUID REFERENCES users(id),
  action VARCHAR(100) NOT NULL,
  resource_type VARCHAR(100),
  resource_id VARCHAR(255),
  details JSONB,
  ip_address INET,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- API keys for external access
CREATE TABLE IF NOT EXISTS api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  key_hash VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255),
  created_by UUID NOT NULL REFERENCES users(id),
  last_used_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP
);

-- Mail server
CREATE TABLE IF NOT EXISTS mailboxes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  email_address VARCHAR(255) NOT NULL UNIQUE,
  display_name VARCHAR(255),
  password_hash VARCHAR(255) NOT NULL,
  storage_quota_mb INT DEFAULT 5120,
  storage_used_mb INT DEFAULT 0,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS email_folders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  mailbox_id UUID NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
  name VARCHAR(100) NOT NULL,
  folder_type VARCHAR(50) DEFAULT 'custom',
  message_count INT DEFAULT 0,
  unseen_count INT DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(mailbox_id, name)
);

CREATE TABLE IF NOT EXISTS emails (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  mailbox_id UUID NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
  folder_id UUID NOT NULL REFERENCES email_folders(id) ON DELETE CASCADE,
  message_id VARCHAR(255) UNIQUE,
  sender_address VARCHAR(255) NOT NULL,
  sender_name VARCHAR(255),
  recipient_addresses TEXT[] NOT NULL,
  cc_addresses TEXT[],
  bcc_addresses TEXT[],
  subject VARCHAR(500),
  body_text TEXT,
  body_html TEXT,
  is_read BOOLEAN DEFAULT false,
  is_flagged BOOLEAN DEFAULT false,
  size_bytes INT,
  headers JSONB,
  received_at TIMESTAMP,
  sent_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS email_attachments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email_id UUID NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
  filename VARCHAR(255) NOT NULL,
  mime_type VARCHAR(100),
  size_bytes INT,
  content_id VARCHAR(255),
  file_path VARCHAR(500),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS email_queue (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email_id UUID REFERENCES emails(id) ON DELETE CASCADE,
  recipient_address VARCHAR(255) NOT NULL,
  status VARCHAR(50) DEFAULT 'pending',
  attempt_count INT DEFAULT 0,
  max_attempts INT DEFAULT 5,
  last_error TEXT,
  next_retry_at TIMESTAMP,
  delivered_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS mail_settings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  mailbox_id UUID NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
  enable_autoreply BOOLEAN DEFAULT false,
  autoreply_subject VARCHAR(255),
  autoreply_body TEXT,
  autoreply_start_at TIMESTAMP,
  autoreply_end_at TIMESTAMP,
  enable_forwarding BOOLEAN DEFAULT false,
  forward_to VARCHAR(255),
  enable_spam_filter BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(mailbox_id)
);

-- Indexes for performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_team_members_team_id ON team_members(team_id);
CREATE INDEX idx_team_members_user_id ON team_members(user_id);
CREATE INDEX idx_nodes_team_id ON nodes(team_id);
CREATE INDEX idx_deployments_team_id ON deployments(team_id);
CREATE INDEX idx_alerts_team_id ON alerts(team_id);
CREATE INDEX idx_alerts_created_at ON alerts(created_at);
CREATE INDEX idx_audit_logs_team_id ON audit_logs(team_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_api_keys_team_id ON api_keys(team_id);
CREATE INDEX idx_mailboxes_user_id ON mailboxes(user_id);
CREATE INDEX idx_mailboxes_email ON mailboxes(email_address);
CREATE INDEX idx_email_folders_mailbox_id ON email_folders(mailbox_id);
CREATE INDEX idx_emails_mailbox_id ON emails(mailbox_id);
CREATE INDEX idx_emails_folder_id ON emails(folder_id);
CREATE INDEX idx_emails_received_at ON emails(received_at);
CREATE INDEX idx_emails_is_read ON emails(is_read);
CREATE INDEX idx_email_attachments_email_id ON email_attachments(email_id);
CREATE INDEX idx_email_queue_status ON email_queue(status);
CREATE INDEX idx_email_queue_next_retry ON email_queue(next_retry_at);
CREATE INDEX idx_mail_settings_mailbox_id ON mail_settings(mailbox_id);
