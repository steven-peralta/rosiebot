import { resolve } from 'url';
import { createLogger, format, transports } from 'winston';
import * as fs from 'fs';
import path from 'path';
import WinstonCloudwatch from 'winston-cloudwatch';

export enum LoggingModule {
  Rosiebot = 'rosiebot',
  Discord = 'discord',
  MWL = 'mwl',
  DB = 'db',
  RandomOrg = 'random.org',
}

const resetColor = '\x1b[0m';

const colors: Record<string, string> = {
  [LoggingModule.Rosiebot]: '\x1b[37m',
  [LoggingModule.Discord]: '\x1b[35m',
  [LoggingModule.MWL]: '\x1b[34m',
  [LoggingModule.DB]: '\x1b[36m',
  [LoggingModule.RandomOrg]: '\x1b[30m',
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

  const logger = createLogger({
    format: format.combine(
      format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss.SSS' }),
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
    exceptionHandlers: [
      // exceptions
      new transports.File({
        filename: path.join(logsDir, 'rosiebot.exceptions.log'),
      }),
    ],
  });

  if (process.env.NODE_ENV === 'staging') {
    logger.add(
      new WinstonCloudwatch({
        logGroupName: 'rosiebot-staging',
        logStreamName: 'rosiebot-staging-info',
        awsRegion: 'us-east-1',
      })
    );
    logger.add(
      new WinstonCloudwatch({
        level: 'error',
        logGroupName: 'rosiebot-staging',
        logStreamName: 'rosiebot-staging-error',
        awsRegion: 'us-east-1',
      })
    );
  }

  return logger;
};

export const logModuleInfo = (
  message: string,
  module: LoggingModule = LoggingModule.Rosiebot
): void => {
  logger.info({ message, module });
};

export const logModuleError = (
  message: string,
  module: LoggingModule = LoggingModule.Rosiebot
): void => {
  logger.error({ message, module });
};

export const logModuleWarning = (
  message: string,
  module: LoggingModule = LoggingModule.Rosiebot
): void => {
  logger.warn({ message, module });
};

const logger = initLogger();

export default logger;
