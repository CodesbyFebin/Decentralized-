import { SMTPServer, SMTPServerOptions, SMTPServerSession } from 'smtp-server'
import { Pool } from 'pg'
import * as bcrypt from 'bcryptjs'
import { MailService } from './mail'
import { v4 as uuidv4 } from 'uuid'

export class MailSMTPServer {
  private smtpServer: SMTPServer
  private pool: Pool
  private mailService: MailService

  constructor(pool: Pool, mailService: MailService) {
    this.pool = pool
    this.mailService = mailService

    const options: SMTPServerOptions = {
      secure: false,
      auth: {
        custom: this.authenticate.bind(this),
      },
      onConnect: this.onConnect.bind(this),
      onMailFrom: this.onMailFrom.bind(this),
      onRcptTo: this.onRcptTo.bind(this),
      onData: this.onData.bind(this),
    }

    this.smtpServer = new SMTPServer(options)
  }

  private async authenticate(auth: any, session: SMTPServerSession, callback: Function) {
    try {
      const mailbox = await this.mailService.getMailboxByEmail(auth.username)
      if (!mailbox) {
        return callback(new Error('User not found'))
      }

      const isValidPassword = await this.mailService.validateMailboxPassword(
        mailbox.id,
        auth.password
      )

      if (!isValidPassword) {
        return callback(new Error('Invalid password'))
      }

      callback(null, { user: mailbox.id })
    } catch (error: any) {
      callback(error)
    }
  }

  private async onConnect(session: SMTPServerSession, callback: Function) {
    console.log(`[SMTP] New connection from ${session.remoteAddress}`)
    callback()
  }

  private async onMailFrom(
    address: any,
    session: SMTPServerSession,
    callback: Function
  ) {
    try {
      const mailbox = await this.mailService.getMailboxByEmail(address.address)
      if (!mailbox || mailbox.id !== (session as any).auth?.user) {
        return callback(new Error('Unauthorized sender'))
      }

      (session as any).mailboxId = mailbox.id
      (session as any).from = address.address
      callback()
    } catch (error: any) {
      callback(error)
    }
  }

  private async onRcptTo(address: any, session: SMTPServerSession, callback: Function) {
    try {
      if (!(session as any).recipients) {
        (session as any).recipients = []
      }
      (session as any).recipients.push(address.address)
      callback()
    } catch (error: any) {
      callback(error)
    }
  }

  private async onData(stream: any, session: SMTPServerSession, callback: Function) {
    try {
      let rawEmail = ''

      stream.on('data', (chunk: Buffer) => {
        rawEmail += chunk.toString()
      })

      stream.on('end', async () => {
        await this.storeEmail(
          (session as any).mailboxId,
          (session as any).from,
          (session as any).recipients,
          rawEmail
        )
        callback()
      })

      stream.on('error', (error: Error) => {
        callback(error)
      })
    } catch (error: any) {
      callback(error)
    }
  }

  private async storeEmail(
    mailboxId: string,
    from: string,
    recipients: string[],
    rawEmail: string
  ) {
    try {
      const mailbox = await this.mailService.getMailbox(mailboxId)
      if (!mailbox) return

      const inboxFolder = await this.mailService.getFolderByName(mailboxId, 'Inbox')
      if (!inboxFolder) return

      // Parse basic email headers (simplified)
      const subjectMatch = rawEmail.match(/^Subject:\s*(.+?)$/m)
      const subject = subjectMatch ? subjectMatch[1] : '(no subject)'

      const bodyMatch = rawEmail.match(/\n\n([\s\S]*)$/m)
      const body = bodyMatch ? bodyMatch[1] : rawEmail

      const emailId = uuidv4()
      const messageId = `<${emailId}@${from.split('@')[1]}>`

      await this.pool.query(
        `INSERT INTO emails
         (id, mailbox_id, folder_id, message_id, sender_address, recipient_addresses, subject, body_text, received_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP)`,
        [
          emailId,
          mailboxId,
          inboxFolder.id,
          messageId,
          from,
          recipients,
          subject,
          body,
        ]
      )

      console.log(`[SMTP] Email stored: ${subject}`)
    } catch (error: any) {
      console.error('[SMTP] Error storing email:', error.message)
    }
  }

  start(port: number = 587) {
    this.smtpServer.listen(port, () => {
      console.log(`📧 SMTP Server listening on port ${port}`)
    })
  }

  stop(callback?: Function) {
    this.smtpServer.close(callback)
  }
}
