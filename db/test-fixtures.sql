-- Test Fixtures for Mail Server Integration Testing
-- Run this after schema.sql to populate test data

-- Test User (password: testpass123 - bcrypt hash)
INSERT INTO users (email, password_hash, full_name, role)
VALUES (
  'testuser@example.com',
  '$2b$10$N9qo8uLOickgx2ZMRZoM.eDaKHVa6Gw.RJpZ0n0dPpKn5gFX2Y9pa',
  'Test User',
  'user'
) ON CONFLICT DO NOTHING;

-- Get the test user ID for use in subsequent inserts
WITH test_user AS (
  SELECT id FROM users WHERE email = 'testuser@example.com'
)
-- Test Team
INSERT INTO teams (name, description, owner_id)
SELECT 'Test Team', 'Team for testing mail functionality', id
FROM test_user
ON CONFLICT DO NOTHING;

-- Test Mailbox 1
INSERT INTO mailboxes (user_id, email_address, display_name, storage_quota_mb)
SELECT id, 'test.mailbox@example.com', 'Test Mailbox', 5120
FROM users WHERE email = 'testuser@example.com'
ON CONFLICT (email_address) DO NOTHING;

-- Test Mailbox 2
INSERT INTO mailboxes (user_id, email_address, display_name, storage_quota_mb)
SELECT id, 'another.mailbox@example.com', 'Another Test Mailbox', 2560
FROM users WHERE email = 'testuser@example.com'
ON CONFLICT (email_address) DO NOTHING;

-- Folders for test mailbox 1 (auto-created by app, but included here for reference)
INSERT INTO email_folders (mailbox_id, name, folder_type, is_system_folder)
SELECT m.id, 'Inbox', 'inbox', true
FROM mailboxes m
WHERE m.email_address = 'test.mailbox@example.com'
  AND NOT EXISTS (
    SELECT 1 FROM email_folders ef
    WHERE ef.mailbox_id = m.id AND ef.name = 'Inbox'
  )
ON CONFLICT DO NOTHING;

INSERT INTO email_folders (mailbox_id, name, folder_type, is_system_folder)
SELECT m.id, 'Sent', 'sent', true
FROM mailboxes m
WHERE m.email_address = 'test.mailbox@example.com'
  AND NOT EXISTS (
    SELECT 1 FROM email_folders ef
    WHERE ef.mailbox_id = m.id AND ef.name = 'Sent'
  )
ON CONFLICT DO NOTHING;

INSERT INTO email_folders (mailbox_id, name, folder_type, is_system_folder)
SELECT m.id, 'Drafts', 'drafts', true
FROM mailboxes m
WHERE m.email_address = 'test.mailbox@example.com'
  AND NOT EXISTS (
    SELECT 1 FROM email_folders ef
    WHERE ef.mailbox_id = m.id AND ef.name = 'Drafts'
  )
ON CONFLICT DO NOTHING;

INSERT INTO email_folders (mailbox_id, name, folder_type, is_system_folder)
SELECT m.id, 'Spam', 'spam', true
FROM mailboxes m
WHERE m.email_address = 'test.mailbox@example.com'
  AND NOT EXISTS (
    SELECT 1 FROM email_folders ef
    WHERE ef.mailbox_id = m.id AND ef.name = 'Spam'
  )
ON CONFLICT DO NOTHING;

INSERT INTO email_folders (mailbox_id, name, folder_type, is_system_folder)
SELECT m.id, 'Trash', 'trash', true
FROM mailboxes m
WHERE m.email_address = 'test.mailbox@example.com'
  AND NOT EXISTS (
    SELECT 1 FROM email_folders ef
    WHERE ef.mailbox_id = m.id AND ef.name = 'Trash'
  )
ON CONFLICT DO NOTHING;

-- Test Email 1: Incoming email
INSERT INTO emails (
  mailbox_id, folder_id, message_id, sender_address,
  recipient_addresses, subject, body_text, body_html,
  is_read, received_at
)
SELECT
  m.id,
  (SELECT id FROM email_folders WHERE mailbox_id = m.id AND name = 'Inbox' LIMIT 1),
  '<test-email-1@example.com>',
  'sender@external.com',
  ARRAY['test.mailbox@example.com'],
  'Welcome to Test Email',
  'This is a test email in plain text.',
  '<html><body><p>This is a test email in HTML.</p></body></html>',
  false,
  CURRENT_TIMESTAMP - INTERVAL '2 hours'
FROM mailboxes m
WHERE m.email_address = 'test.mailbox@example.com'
  AND NOT EXISTS (
    SELECT 1 FROM emails e
    WHERE e.message_id = '<test-email-1@example.com>'
  )
ON CONFLICT DO NOTHING;

-- Test Email 2: Already read email
INSERT INTO emails (
  mailbox_id, folder_id, message_id, sender_address,
  recipient_addresses, subject, body_text, body_html,
  is_read, received_at
)
SELECT
  m.id,
  (SELECT id FROM email_folders WHERE mailbox_id = m.id AND name = 'Inbox' LIMIT 1),
  '<test-email-2@example.com>',
  'colleague@work.com',
  ARRAY['test.mailbox@example.com'],
  'Project Update',
  'Here is the status update for the project.',
  '<html><body><p>Here is the status update for the project.</p></body></html>',
  true,
  CURRENT_TIMESTAMP - INTERVAL '1 day'
FROM mailboxes m
WHERE m.email_address = 'test.mailbox@example.com'
  AND NOT EXISTS (
    SELECT 1 FROM emails e
    WHERE e.message_id = '<test-email-2@example.com>'
  )
ON CONFLICT DO NOTHING;

-- Test Email 3: Sent email
INSERT INTO emails (
  mailbox_id, folder_id, message_id, sender_address,
  recipient_addresses, subject, body_text, body_html,
  is_read, received_at
)
SELECT
  m.id,
  (SELECT id FROM email_folders WHERE mailbox_id = m.id AND name = 'Sent' LIMIT 1),
  '<test-email-3@example.com>',
  'test.mailbox@example.com',
  ARRAY['recipient@example.com'],
  'Response to Inquiry',
  'Thank you for your inquiry. Here is my response.',
  '<html><body><p>Thank you for your inquiry. Here is my response.</p></body></html>',
  true,
  CURRENT_TIMESTAMP - INTERVAL '6 hours'
FROM mailboxes m
WHERE m.email_address = 'test.mailbox@example.com'
  AND NOT EXISTS (
    SELECT 1 FROM emails e
    WHERE e.message_id = '<test-email-3@example.com>'
  )
ON CONFLICT DO NOTHING;

-- Test Email with Attachment
INSERT INTO emails (
  mailbox_id, folder_id, message_id, sender_address,
  recipient_addresses, subject, body_text, body_html,
  is_read, received_at
)
SELECT
  m.id,
  (SELECT id FROM email_folders WHERE mailbox_id = m.id AND name = 'Inbox' LIMIT 1),
  '<test-email-attachment@example.com>',
  'documents@company.com',
  ARRAY['test.mailbox@example.com'],
  'Important Documents',
  'Please find the attached documents.',
  '<html><body><p>Please find the attached documents.</p></body></html>',
  false,
  CURRENT_TIMESTAMP - INTERVAL '12 hours'
FROM mailboxes m
WHERE m.email_address = 'test.mailbox@example.com'
  AND NOT EXISTS (
    SELECT 1 FROM emails e
    WHERE e.message_id = '<test-email-attachment@example.com>'
  )
ON CONFLICT DO NOTHING;

-- Test Attachment
INSERT INTO email_attachments (
  email_id, filename, mime_type, size_bytes, file_checksum
)
SELECT
  e.id,
  'document.pdf',
  'application/pdf',
  1024567,
  'sha256_abc123def456'
FROM emails e
WHERE e.message_id = '<test-email-attachment@example.com>'
  AND NOT EXISTS (
    SELECT 1 FROM email_attachments ea
    WHERE ea.email_id = e.id AND ea.filename = 'document.pdf'
  )
ON CONFLICT DO NOTHING;

-- Mail Settings for test mailbox
INSERT INTO mail_settings (
  mailbox_id, auto_reply_enabled, auto_reply_message,
  spam_filter_enabled, encryption_enabled
)
SELECT
  id,
  false,
  'I am currently out of the office. I will reply when I return.',
  true,
  true
FROM mailboxes m
WHERE m.email_address = 'test.mailbox@example.com'
  AND NOT EXISTS (
    SELECT 1 FROM mail_settings ms
    WHERE ms.mailbox_id = m.id
  )
ON CONFLICT DO NOTHING;

-- Test Email Queue (for outbound emails)
INSERT INTO email_queue (
  mailbox_id, recipient_address, subject, status,
  retry_attempts, next_retry_at, created_at
)
SELECT
  id,
  'queue-test@example.com',
  'Test Queue Email',
  'queued',
  0,
  CURRENT_TIMESTAMP + INTERVAL '5 minutes',
  CURRENT_TIMESTAMP
FROM mailboxes m
WHERE m.email_address = 'test.mailbox@example.com'
  AND NOT EXISTS (
    SELECT 1 FROM email_queue eq
    WHERE eq.mailbox_id = m.id
      AND eq.subject = 'Test Queue Email'
  )
ON CONFLICT DO NOTHING;

-- Display test data summary
SELECT 'TEST DATA SETUP COMPLETE' AS status;
SELECT COUNT(*) AS users FROM users;
SELECT COUNT(*) AS teams FROM teams;
SELECT COUNT(*) AS mailboxes FROM mailboxes;
SELECT COUNT(*) AS emails FROM emails;
SELECT COUNT(*) AS attachments FROM email_attachments;
SELECT COUNT(*) AS queued_emails FROM email_queue WHERE status = 'queued';
