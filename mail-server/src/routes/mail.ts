import express, { Request, Response } from 'express';
import { MailService } from '../services/mail';
import { authMiddleware } from '../middleware/auth';
import { validateRequest } from '../middleware/validate';
import Joi from 'joi';

const router = express.Router();
const mailService = new MailService();

router.post('/mailboxes', validateRequest(Joi.object({
  username: Joi.string().required(),
  password: Joi.string().min(8).required(),
  name: Joi.string().required(),
})), async (req: Request, res: Response) => {
  try {
    const result = await mailService.createMailbox(req.body.username, req.body.password, req.body.name);
    res.status(201).json({ success: true, data: result });
  } catch (err: any) {
    res.status(400).json({ success: false, error: err.message });
  }
});

router.get('/mailboxes/:id', authMiddleware, async (req: Request, res: Response) => {
  try {
    const mailbox = await mailService.getMailbox(req.params.id);
    res.json({ success: true, data: mailbox });
  } catch (err: any) {
    res.status(404).json({ success: false, error: err.message });
  }
});

router.put('/mailboxes/:id', authMiddleware, async (req: Request, res: Response) => {
  try {
    const result = await mailService.updateMailbox(req.params.id, req.body);
    res.json({ success: true, data: result });
  } catch (err: any) {
    res.status(400).json({ success: false, error: err.message });
  }
});

router.delete('/mailboxes/:id', authMiddleware, async (req: Request, res: Response) => {
  try {
    await mailService.deleteMailbox(req.params.id);
    res.json({ success: true });
  } catch (err: any) {
    res.status(400).json({ success: false, error: err.message });
  }
});

router.get('/mailboxes/:mailboxId/emails', authMiddleware, async (req: Request, res: Response) => {
  try {
    const limit = parseInt(req.query.limit as string) || 20;
    const offset = parseInt(req.query.offset as string) || 0;
    const emails = await mailService.listEmails(req.params.mailboxId, limit, offset);
    res.json({ success: true, data: emails });
  } catch (err: any) {
    res.status(400).json({ success: false, error: err.message });
  }
});

router.get('/emails/:id', authMiddleware, async (req: Request, res: Response) => {
  try {
    const email = await mailService.getEmail(req.params.id);
    res.json({ success: true, data: email });
  } catch (err: any) {
    res.status(404).json({ success: false, error: err.message });
  }
});

router.patch('/emails/:id', authMiddleware, async (req: Request, res: Response) => {
  try {
    const email = await mailService.updateEmail(req.params.id, req.body);
    res.json({ success: true, data: email });
  } catch (err: any) {
    res.status(400).json({ success: false, error: err.message });
  }
});

router.delete('/emails/:id', authMiddleware, async (req: Request, res: Response) => {
  try {
    await mailService.deleteEmail(req.params.id);
    res.json({ success: true });
  } catch (err: any) {
    res.status(400).json({ success: false, error: err.message });
  }
});

router.get('/mailboxes/:mailboxId/folders', authMiddleware, async (req: Request, res: Response) => {
  try {
    const folders = await mailService.listFolders(req.params.mailboxId);
    res.json({ success: true, data: folders });
  } catch (err: any) {
    res.status(400).json({ success: false, error: err.message });
  }
});

router.post('/mailboxes/:mailboxId/folders', authMiddleware, validateRequest(Joi.object({
  name: Joi.string().required(),
})), async (req: Request, res: Response) => {
  try {
    const folder = await mailService.createFolder(req.params.mailboxId, req.body.name);
    res.status(201).json({ success: true, data: folder });
  } catch (err: any) {
    res.status(400).json({ success: false, error: err.message });
  }
});

router.get('/queue/status', authMiddleware, async (req: Request, res: Response) => {
  try {
    const status = await mailService.getQueueStatus();
    res.json({ success: true, data: status });
  } catch (err: any) {
    res.status(400).json({ success: false, error: err.message });
  }
});

export default router;
