import { pool } from '../index';
import bcrypt from 'bcryptjs';
import jwt from 'jsonwebtoken';
import { v4 as uuidv4 } from 'uuid';

export class MailService {
  async createMailbox(username: string, password: string, name: string) {
    const hashedPassword = await bcrypt.hash(password, 10);
    const mailboxId = uuidv4();
    const result = await pool.query(
      `INSERT INTO mailboxes (id, username, password_hash, name, created_at)
       VALUES ($1, $2, $3, $4, $5)
       RETURNING id, username, name, created_at`,
      [mailboxId, username, hashedPassword, name, new Date().toISOString()]
    );
    const token = jwt.sign(
      { mailboxId, username },
      process.env.JWT_SECRET || 'dev-secret-key',
      { expiresIn: process.env.JWT_EXPIRY || '24h' } as any
    );
    return { ...result.rows[0], token };
  }

  async authenticateMailbox(username: string, password: string) {
    const result = await pool.query(
      'SELECT id, password_hash FROM mailboxes WHERE username = $1',
      [username]
    );
    if (result.rows.length === 0) throw new Error('Mailbox not found');
    const mailbox = result.rows[0];
    const passwordMatch = await bcrypt.compare(password, mailbox.password_hash);
    if (!passwordMatch) throw new Error('Invalid password');
    const token = jwt.sign(
      { mailboxId: mailbox.id, username },
      process.env.JWT_SECRET || 'dev-secret-key',
      { expiresIn: process.env.JWT_EXPIRY || '24h' } as any
    );
    return { mailboxId: mailbox.id, token };
  }

  async getMailbox(id: string) {
    const result = await pool.query(
      'SELECT id, username, name, created_at FROM mailboxes WHERE id = $1',
      [id]
    );
    if (result.rows.length === 0) throw new Error('Mailbox not found');
    return result.rows[0];
  }

  async updateMailbox(id: string, updates: any) {
    const fields: string[] = [];
    const values: any[] = [id];
    let paramCount = 2;
    if (updates.name) {
      fields.push(`name = $${paramCount++}`);
      values.push(updates.name);
    }
    if (updates.password) {
      const hashedPassword = await bcrypt.hash(updates.password, 10);
      fields.push(`password_hash = $${paramCount++}`);
      values.push(hashedPassword);
    }
    if (fields.length === 0) return this.getMailbox(id);
    const query = `UPDATE mailboxes SET ${fields.join(', ')} WHERE id = $1 RETURNING id, username, name, created_at`;
    const result = await pool.query(query, values);
    return result.rows[0];
  }

  async deleteMailbox(id: string) {
    await pool.query('DELETE FROM mailboxes WHERE id = $1', [id]);
  }

  async listEmails(mailboxId: string, limit: number, offset: number) {
    const result = await pool.query(
      `SELECT id, mailbox_id, sender, subject, body, state, flagged, read, received_at
       FROM emails WHERE mailbox_id = $1 ORDER BY received_at DESC LIMIT $2 OFFSET $3`,
      [mailboxId, limit, offset]
    );
    return result.rows;
  }

  async getEmail(id: string) {
    const result = await pool.query('SELECT * FROM emails WHERE id = $1', [id]);
    if (result.rows.length === 0) throw new Error('Email not found');
    return result.rows[0];
  }

  async updateEmail(id: string, updates: any) {
    const fields: string[] = [];
    const values: any[] = [id];
    let paramCount = 2;
    if (updates.state) {
      fields.push(`state = $${paramCount++}`);
      values.push(updates.state);
    }
    if (typeof updates.flagged !== 'undefined') {
      fields.push(`flagged = $${paramCount++}`);
      values.push(updates.flagged);
    }
    if (typeof updates.read !== 'undefined') {
      fields.push(`read = $${paramCount++}`);
      values.push(updates.read);
    }
    if (fields.length === 0) return this.getEmail(id);
    const query = `UPDATE emails SET ${fields.join(', ')} WHERE id = $1 RETURNING *`;
    const result = await pool.query(query, values);
    return result.rows[0];
  }

  async deleteEmail(id: string) {
    await pool.query('DELETE FROM emails WHERE id = $1', [id]);
  }

  async listFolders(mailboxId: string) {
    const result = await pool.query(
      'SELECT * FROM folders WHERE mailbox_id = $1 ORDER BY name', [mailboxId]
    );
    return result.rows;
  }

  async createFolder(mailboxId: string, name: string) {
    const folderId = uuidv4();
    const result = await pool.query(
      `INSERT INTO folders (id, mailbox_id, name) VALUES ($1, $2, $3) RETURNING *`,
      [folderId, mailboxId, name]
    );
    return result.rows[0];
  }

  async getQueueStatus() {
    const result = await pool.query(
      `SELECT COUNT(*) FILTER (WHERE state = 'pending') as pending, COUNT(*) FILTER (WHERE state = 'sent') as sent FROM queue`
    );
    return result.rows[0];
  }

  async storeEmail(mailboxId: string, sender: string, subject: string, body: string) {
    const emailId = uuidv4();
    const result = await pool.query(
      `INSERT INTO emails (id, mailbox_id, sender, subject, body, state, received_at)
       VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *`,
      [emailId, mailboxId, sender, subject, body, 'received', new Date().toISOString()]
    );
    return result.rows[0];
  }
}
