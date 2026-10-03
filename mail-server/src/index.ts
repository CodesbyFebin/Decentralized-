import express from 'express';
import cors from 'cors';
import dotenv from 'dotenv';
import { Pool } from 'pg';
import mailRoutes from './routes/mail';
import { SMTPServer } from './services/smtp-server';

dotenv.config();

const app = express();
const PORT = process.env.PORT || 3001;

export const pool = new Pool({
  host: process.env.DB_HOST,
  port: parseInt(process.env.DB_PORT || '5432'),
  user: process.env.DB_USER,
  password: process.env.DB_PASSWORD,
  database: process.env.DB_NAME,
});

app.use(cors());
app.use(express.json({ limit: '50mb' }));
app.use(express.urlencoded({ limit: '50mb', extended: true }));

app.get('/api/v1/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

app.use('/api/v1', mailRoutes);

app.use((err: any, req: express.Request, res: express.Response, next: express.NextFunction) => {
  console.error('Error:', err);
  res.status(err.status || 500).json({
    success: false,
    error: err.message || 'Internal server error',
  });
});

app.listen(PORT, () => {
  console.log(`📧 Mail Server running on port ${PORT}`);
  console.log(`API: http://localhost:${PORT}/api/v1`);
});

const smtpPort = parseInt(process.env.MAIL_SMTP_PORT || '2525');
const smtpServer = new SMTPServer();
try {
  smtpServer.start(smtpPort, 587);
} catch (err) {
  console.warn('Failed to start SMTP server:', err);
}

process.on('SIGTERM', () => {
  console.log('Shutting down...');
  smtpServer.stop();
  pool.end();
  process.exit(0);
});
