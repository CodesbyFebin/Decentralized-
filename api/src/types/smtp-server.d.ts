declare module 'smtp-server' {
  import { Socket } from 'net';

  export interface SMTPServerOptions {
    secure?: boolean;
    auth?: {
      custom?: (auth: any, session: SMTPServerSession, callback: (err?: Error | null, user?: any) => void) => void;
    };
    onConnect?: (session: SMTPServerSession, callback: (err?: Error | null) => void) => void;
    onMailFrom?: (address: any, session: SMTPServerSession, callback: (err?: Error | null) => void) => void;
    onRcptTo?: (address: any, session: SMTPServerSession, callback: (err?: Error | null) => void) => void;
    onData?: (stream: NodeJS.ReadableStream, session: SMTPServerSession, callback: (err?: Error | null) => void) => void;
  }

  export interface SMTPServerSession {
    remoteAddress?: string;
    remotePort?: number;
    clientHostname?: string;
    openingCommand?: string;
    secure?: boolean;
    auth?: {
      user: string;
    };
  }

  export class SMTPServer {
    constructor(options: SMTPServerOptions);
    listen(port: number, callback?: () => void): void;
    close(callback?: () => void): void;
  }
}
