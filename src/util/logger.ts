import { createLogger, format, transports } from 'winston';
import { resolve } from 'url';
import * as fs from 'fs';
import path from 'path';

const resetColor = '\x1b[0m';
const colors: Record<string, string> = {
  rosiebot: '\x1b[37m',
  discord: '\x1b[35m',
  mwl: '\x1b[34m',
  db: '\x1b[36m',
  random: '\x1b[30m',
};

export const logsDir = resolve(resolve(__dirname, '..'), 'logs');

const logFormatter = (info: Record<string, string>) =>
  info.module
    ? `${info.timestamp} [${format
        .colorize()
        .colorize(info.level, info.level.toUpperCase())}] ${
        colors[info.module]
      }[${info.module}]${resetColor}: ${info.message}`
    : `${info.timestamp} [${format
        .colorize()
        .colorize(info.level, info.level.toUpperCase())}]: ${info.message}`;

const initLogger = () => {
  if (!fs.existsSync(logsDir)) {
    fs.mkdirSync(logsDir);
  }

  return createLogger({
    format: format.combine(
      format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
      format.printf(logFormatter)
    ),
    transports: [
      new transports.Console(),
      // combined
      new transports.File({
        filename: path.join(logsDir, 'rosiebot.log'),
      }),
      // error
      new transports.File({
        filename: path.join(logsDir, 'rosiebot.error.log'),
        level: 'error',
      }),
    ],
  });
};

export const logInfo = (message: any, module = 'rosiebot'): void => {
  logger.info({ message, module });
};

export const logError = (message: any, module = 'rosiebot'): void => {
  logger.error({ message, module });
};

export const logWarn = (message: any, module = 'rosiebot'): void => {
  logger.warn({ message, module });
};

const logger = initLogger();

export default logger;
