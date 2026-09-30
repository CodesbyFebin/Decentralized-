import { Pool } from 'pg'
import { v4 as uuidv4 } from 'uuid'
import * as bcrypt from 'bcryptjs'
import nodemailer from 'nodemailer'

export interface Mailbox {
  id: string
  user_id: string
  email_address: string
  display_name?: string
  storage_quota_mb: number
  storage_used_mb: number
  is_active: boolean
  created_at: Date
  updated_at: Date
}

export interface EmailMessage {
  id: string
  mailbox_id: string
  folder_id: string
  message_id?: string
  sender_address: string
  sender_name?: string
  recipient_addresses: string[]
  cc_addresses?: string[]
  bcc_addresses?: string[]
  subject?: string
  body_text?: string
  body_html?: string
  is_read: boolean
  is_flagged: boolean
  size_bytes?: number
  received_at?: Date
  sent_at?: Date
  created_at: Date
  updated_at: Date
}

export interface MailFolder {
  id: string
  mailbox_id: string
  name: string
  folder_type: string
  message_count: number
  unseen_count: number
  created_at: Date
}

export class MailService {
  private pool: Pool
  private smtpTransporter: any

  constructor(pool: Pool) {
    this.pool = pool
    this.initializeTransporter()
  }

  private initializeTransporter() {
    this.smtpTransporter = nodemailer.createTransport({
      host: process.env.SMTP_HOST || 'localhost',
      port: parseInt(process.env.SMTP_PORT || '587'),
      secure: process.env.SMTP_SECURE === 'true',
      auth: {
        user: process.env.SMTP_USER,
        pass: process.env.SMTP_PASSWORD,
      },
    })
  }

  async createMailbox(
    userId: string,
    emailAddress: string,
    displayName?: string,
    password?: string
  ): Promise<Mailbox> {
    const id = uuidv4()
    const hashedPassword = password
      ? await bcrypt.hash(password, 10)
      : await bcrypt.hash(Math.random().toString(), 10)

    const result = await this.pool.query(
      `INSERT INTO mailboxes (id, user_id, email_address, display_name, password_hash)
       VALUES ($1, $2, $3, $4, $5)
       RETURNING *`,
      [id, userId, emailAddress, displayName, hashedPassword]
    )

    await this.initializeMailboxFolders(id)
    return result.rows[0]
  }

  private async initializeMailboxFolders(mailboxId: string) {
    const folders = ['Inbox', 'Sent', 'Drafts', 'Trash', 'Spam']
    for (const folderName of folders) {
      await this.pool.query(
        `INSERT INTO email_folders (id, mailbox_id, name, folder_type)
         VALUES ($1, $2, $3, $4)`,
        [uuidv4(), mailboxId, folderName, folderName.toLowerCase()]
      )
    }
  }

  async getMailbox(mailboxId: string): Promise<Mailbox | null> {
    const result = await this.pool.query(
      'SELECT * FROM mailboxes WHERE id = $1',
      [mailboxId]
    )
    return result.rows[0] || null
  }

  async getMailboxByEmail(emailAddress: string): Promise<Mailbox | null> {
    const result = await this.pool.query(
      'SELECT * FROM mailboxes WHERE email_address = $1',
      [emailAddress]
    )
    return result.rows[0] || null
  }

  async getFolders(mailboxId: string): Promise<MailFolder[]> {
    const result = await this.pool.query(
      'SELECT * FROM email_folders WHERE mailbox_id = $1 ORDER BY name',
      [mailboxId]
    )
    return result.rows
  }

  async getFolderByName(mailboxId: string, folderName: string): Promise<MailFolder | null> {
    const result = await this.pool.query(
      'SELECT * FROM email_folders WHERE mailbox_id = $1 AND name = $2',
      [mailboxId, folderName]
    )
    return result.rows[0] || null
  }

  async getEmails(mailboxId: string, folderId: string, limit = 50, offset = 0): Promise<EmailMessage[]> {
    const result = await this.pool.query(
      `SELECT * FROM emails
       WHERE mailbox_id = $1 AND folder_id = $2
       ORDER BY received_at DESC NULLS LAST
       LIMIT $3 OFFSET $4`,
      [mailboxId, folderId, limit, offset]
    )
    return result.rows
  }

  async getEmail(emailId: string): Promise<EmailMessage | null> {
    const result = await this.pool.query(
      'SELECT * FROM emails WHERE id = $1',
      [emailId]
    )
    return result.rows[0] || null
  }

  async sendEmail(
    fromMailboxId: string,
    toAddresses: string[],
    ccAddresses?: string[],
    bccAddresses?: string[],
    subject?: string,
    bodyText?: string,
    bodyHtml?: string
  ): Promise<EmailMessage> {
    const mailbox = await this.getMailbox(fromMailboxId)
    if (!mailbox) throw new Error('Mailbox not found')

    const emailId = uuidv4()
    const messageId = `<${emailId}@${mailbox.email_address.split('@')[1]}>`

    await this.pool.query(
      `INSERT INTO emails
       (id, mailbox_id, folder_id, message_id, sender_address, sender_name,
        recipient_addresses, cc_addresses, bcc_addresses, subject, body_text, body_html, sent_at)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, CURRENT_TIMESTAMP)`,
      [
        emailId,
        fromMailboxId,
        (await this.getFolderByName(fromMailboxId, 'Sent'))?.id,
        messageId,
        mailbox.email_address,
        mailbox.display_name,
        toAddresses,
        ccAddresses || [],
        bccAddresses || [],
        subject,
        bodyText,
        bodyHtml,
      ]
    )

    await this.queueEmailForDelivery(emailId, toAddresses)

    return this.getEmail(emailId) as Promise<EmailMessage>
  }

  private async queueEmailForDelivery(emailId: string, recipients: string[]) {
    for (const recipient of recipients) {
      await this.pool.query(
        `INSERT INTO email_queue (id, email_id, recipient_address, status)
         VALUES ($1, $2, $3, 'pending')`,
        [uuidv4(), emailId, recipient]
      )
    }
  }

  async markEmailAsRead(emailId: string, isRead = true): Promise<void> {
    await this.pool.query(
      'UPDATE emails SET is_read = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2',
      [isRead, emailId]
    )
  }

  async moveEmailToFolder(emailId: string, folderId: string): Promise<void> {
    await this.pool.query(
      'UPDATE emails SET folder_id = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2',
      [folderId, emailId]
    )
  }

  async deleteEmail(emailId: string): Promise<void> {
    const email = await this.getEmail(emailId)
    if (!email) throw new Error('Email not found')

    const trashFolder = await this.getFolderByName(email.mailbox_id, 'Trash')
    if (trashFolder) {
      await this.moveEmailToFolder(emailId, trashFolder.id)
    } else {
      await this.pool.query('DELETE FROM emails WHERE id = $1', [emailId])
    }
  }

  async processEmailQueue() {
    const queuedEmails = await this.pool.query(
      `SELECT * FROM email_queue
       WHERE status = 'pending' AND (next_retry_at IS NULL OR next_retry_at <= CURRENT_TIMESTAMP)
       AND attempt_count < max_attempts
       ORDER BY created_at ASC
       LIMIT 10`
    )

    for (const item of queuedEmails.rows) {
      await this.deliverEmail(item)
    }
  }

  private async deliverEmail(queueItem: any) {
    try {
      const email = await this.getEmail(queueItem.email_id)
      if (!email) return

      await this.smtpTransporter.sendMail({
        from: email.sender_address,
        to: queueItem.recipient_address,
        cc: email.cc_addresses?.join(','),
        subject: email.subject,
        text: email.body_text,
        html: email.body_html,
        messageId: email.message_id,
      })

      await this.pool.query(
        `UPDATE email_queue SET status = 'delivered', delivered_at = CURRENT_TIMESTAMP
         WHERE id = $1`,
        [queueItem.id]
      )
    } catch (error: any) {
      const nextRetryAt = new Date(Date.now() + (1000 * 60 * 5)) // 5 minutes
      await this.pool.query(
        `UPDATE email_queue
         SET attempt_count = attempt_count + 1,
             last_error = $1,
             next_retry_at = $2,
             updated_at = CURRENT_TIMESTAMP
         WHERE id = $3`,
        [error.message, nextRetryAt, queueItem.id]
      )
    }
  }

  async validateMailboxPassword(mailboxId: string, password: string): Promise<boolean> {
    const result = await this.pool.query(
      'SELECT password_hash FROM mailboxes WHERE id = $1',
      [mailboxId]
    )
    if (!result.rows[0]) return false
    return bcrypt.compare(password, result.rows[0].password_hash)
  }
}
