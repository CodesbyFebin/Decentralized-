INSERT INTO mailboxes (id, username, password_hash, name) VALUES
  ('10000000-0000-0000-0000-000000000001', 'alice', '$2a$10$N9qo8uLOickgx2ZMRZoMye0Z8s4FzJeRb5VUJfpUw0MfXDC7XcB8C', 'Alice User'),
  ('10000000-0000-0000-0000-000000000002', 'bob', '$2a$10$R9qo8uLOickgx2ZMRZoMye0Z8s4FzJeRb5VUJfpUw0MfXDC7XcB8D', 'Bob User');

INSERT INTO emails (id, mailbox_id, sender, subject, body, state, received_at) VALUES
  ('30000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'bob@example.com', 'Hello', 'Test email', 'received', NOW());
