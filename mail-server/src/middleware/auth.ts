import express from 'express';
import jwt from 'jsonwebtoken';

export const authMiddleware = (req: express.Request, res: express.Response, next: express.NextFunction) => {
  const token = req.headers.authorization?.split(' ')[1];
  if (!token) {
    return res.status(401).json({ success: false, error: 'Missing token' });
  }
  try {
    const decoded = jwt.verify(token, process.env.JWT_SECRET || 'dev-secret-key') as any;
    (req as any).user = decoded;
    next();
  } catch (err: any) {
    res.status(401).json({ success: false, error: 'Invalid token' });
  }
};
