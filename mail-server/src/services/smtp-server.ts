import { SMTPServer as NodeSMTPServer, SMTPServerSession } from 'smtp-server';
import { MailService } from './mail';

const mailService = new MailService();

export class SMTPServer {
  private server: NodeSMTPServer | null = null;

  start(port: number, tlsPort: number) {
    const smtpOptions = {
      secure: false,
      authOptional: true,
      logger: false,
      onData: this.handleData.bind(this),
      onAuth: this.handleAuth.bind(this),
    };
    this.server = new NodeSMTPServer(smtpOptions);
    this.server.listen(port, () => {
      console.log(`📧 SMTP Server running on port ${port}`);
    }).on('error', (err: any) => {
      if (err.code === 'EACCES') {
        console.warn(`⚠️  Cannot bind to port ${port} (requires root). SMTP disabled.`);
      } else {
        console.error('SMTP Error:', err);
      }
    });
  }

  private async handleAuth(auth: any, session: SMTPServerSession, callback: Function) {
    try {
      const result = await mailService.authenticateMailbox(auth.username, auth.password);
      callback(null, { user: result.mailboxId });
    } catch (err: any) {
      callback(new Error(err.message));
    }
  }

  private async handleData(stream: any, session: SMTPServerSession, callback: Function) {
    try {
      const chunks: Buffer[] = [];
      stream.on('data', (chunk: Buffer) => chunks.push(chunk));
      stream.on('end', async () => {
        const message = Buffer.concat(chunks).toString('utf8');
        const [headers] = message.split('\n\n');
        let from = 'unknown@localhost', subject = '(no subject)';
        for (const line of headers.split('\n')) {
          if (line.startsWith('From:')) from = line.substring(5).trim();
          if (line.startsWith('Subject:')) subject = line.substring(8).trim();
        }
        await mailService.storeEmail(session.user as string, from, subject, message);
        callback();
      });
    } catch (err: any) {
      callback(err);
    }
  }

  stop() {
    if (this.server) this.server.close();
  }
}
