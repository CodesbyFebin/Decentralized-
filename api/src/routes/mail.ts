import { Router, Request, Response } from 'express'
import { Pool } from 'pg'
import { MailService } from '../services/mail'
import { authenticateToken } from '../middleware/auth'
import Joi from 'joi'

export function createMailRoutes(pool: Pool): Router {
  const router = Router()
  const mailService = new MailService(pool)

  // Validation schemas
  const createMailboxSchema = Joi.object({
    email_address: Joi.string().email().required(),
    display_name: Joi.string().optional(),
    password: Joi.string().min(8).optional(),
  })

  const sendEmailSchema = Joi.object({
    to_addresses: Joi.array().items(Joi.string().email()).required(),
    cc_addresses: Joi.array().items(Joi.string().email()).optional(),
    bcc_addresses: Joi.array().items(Joi.string().email()).optional(),
    subject: Joi.string().optional(),
    body_text: Joi.string().optional(),
    body_html: Joi.string().optional(),
  })

  // Create mailbox
  router.post('/mailbox', authenticateToken, async (req: Request, res: Response) => {
    try {
      const { error, value } = createMailboxSchema.validate(req.body)
      if (error) {
        return res.status(400).json({ success: false, error: error.details[0].message })
      }

      const mailbox = await mailService.createMailbox(
        (req as any).user.id,
        value.email_address,
        value.display_name,
        value.password
      )

      res.status(201).json({
        success: true,
        data: mailbox,
      })
    } catch (err: any) {
      res.status(500).json({
        success: false,
        error: err.message || 'Failed to create mailbox',
      })
    }
  })

  // Get mailbox
  router.get('/mailbox/:id', authenticateToken, async (req: Request, res: Response) => {
    try {
      const mailbox = await mailService.getMailbox(req.params.id)
      if (!mailbox) {
        return res.status(404).json({ success: false, error: 'Mailbox not found' })
      }

      res.json({ success: true, data: mailbox })
    } catch (err: any) {
      res.status(500).json({
        success: false,
        error: err.message || 'Failed to fetch mailbox',
      })
    }
  })

  // Get folders
  router.get('/mailbox/:mailboxId/folders', authenticateToken, async (req: Request, res: Response) => {
    try {
      const folders = await mailService.getFolders(req.params.mailboxId)
      res.json({ success: true, data: folders })
    } catch (err: any) {
      res.status(500).json({
        success: false,
        error: err.message || 'Failed to fetch folders',
      })
    }
  })

  // Get emails in folder
  router.get(
    '/mailbox/:mailboxId/folder/:folderId/emails',
    authenticateToken,
    async (req: Request, res: Response) => {
      try {
        const limit = Math.min(parseInt(req.query.limit as string) || 50, 100)
        const offset = parseInt(req.query.offset as string) || 0

        const emails = await mailService.getEmails(
          req.params.mailboxId,
          req.params.folderId,
          limit,
          offset
        )

        res.json({ success: true, data: emails })
      } catch (err: any) {
        res.status(500).json({
          success: false,
          error: err.message || 'Failed to fetch emails',
        })
      }
    }
  )

  // Get email details
  router.get('/email/:id', authenticateToken, async (req: Request, res: Response) => {
    try {
      const email = await mailService.getEmail(req.params.id)
      if (!email) {
        return res.status(404).json({ success: false, error: 'Email not found' })
      }

      res.json({ success: true, data: email })
    } catch (err: any) {
      res.status(500).json({
        success: false,
        error: err.message || 'Failed to fetch email',
      })
    }
  })

  // Send email
  router.post('/send', authenticateToken, async (req: Request, res: Response) => {
    try {
      const { error, value } = sendEmailSchema.validate(req.body)
      if (error) {
        return res.status(400).json({ success: false, error: error.details[0].message })
      }

      // Get mailbox for user
      const mailboxes = await (pool as any).query(
        'SELECT id FROM mailboxes WHERE user_id = $1 LIMIT 1',
        [(req as any).user.id]
      )

      if (mailboxes.rows.length === 0) {
        return res.status(400).json({ success: false, error: 'No mailbox configured' })
      }

      const email = await mailService.sendEmail(
        mailboxes.rows[0].id,
        value.to_addresses,
        value.cc_addresses,
        value.bcc_addresses,
        value.subject,
        value.body_text,
        value.body_html
      )

      res.status(201).json({
        success: true,
        data: email,
      })
    } catch (err: any) {
      res.status(500).json({
        success: false,
        error: err.message || 'Failed to send email',
      })
    }
  })

  // Mark email as read
  router.patch('/email/:id/read', authenticateToken, async (req: Request, res: Response) => {
    try {
      const isRead = req.body.is_read !== false
      await mailService.markEmailAsRead(req.params.id, isRead)

      res.json({
        success: true,
        message: `Email marked as ${isRead ? 'read' : 'unread'}`,
      })
    } catch (err: any) {
      res.status(500).json({
        success: false,
        error: err.message || 'Failed to update email',
      })
    }
  })

  // Move email to folder
  router.patch('/email/:id/folder', authenticateToken, async (req: Request, res: Response) => {
    try {
      const { folder_id } = req.body
      if (!folder_id) {
        return res.status(400).json({ success: false, error: 'folder_id required' })
      }

      await mailService.moveEmailToFolder(req.params.id, folder_id)

      res.json({
        success: true,
        message: 'Email moved to folder',
      })
    } catch (err: any) {
      res.status(500).json({
        success: false,
        error: err.message || 'Failed to move email',
      })
    }
  })

  // Delete email
  router.delete('/email/:id', authenticateToken, async (req: Request, res: Response) => {
    try {
      await mailService.deleteEmail(req.params.id)

      res.json({
        success: true,
        message: 'Email deleted',
      })
    } catch (err: any) {
      res.status(500).json({
        success: false,
        error: err.message || 'Failed to delete email',
      })
    }
  })

  return router
}
