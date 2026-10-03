declare module 'smtp-server' {
  export interface SMTPServerOptions {
    [key: string]: any;
  }

  export class SMTPServer {
    constructor(options?: SMTPServerOptions);
    listen(port: number, host?: string, callback?: () => void): void;
    close(callback?: () => void): void;
  }
}
