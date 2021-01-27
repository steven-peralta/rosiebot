import { Connection } from 'mongoose';
import {
  LoggingModule,
  logModuleError,
  logModuleInfo,
} from 'rosiebot/src/util/logger';

const hookEvents = (db: Connection): void => {
  db.on('connecting', () => {
    logModuleInfo('Connecting to MongoDB...', LoggingModule.DB);
  });

  db.on('connected', () => {
    logModuleInfo('Connected to MongoDB', LoggingModule.DB);
  });

  db.on('disconnecting', () => {
    logModuleInfo('Disconnecting from MongoDB...', LoggingModule.DB);
  });

  db.on('close', () => {
    logModuleInfo('Disconnected from MongoDB', LoggingModule.DB);
  });

  db.on('disconnected', () => {
    logModuleError('Lost connection to MongoDB', LoggingModule.DB);
  });

  db.on('reconnectFailed', () => {
    logModuleError('Reconnect to MongoDB failed', LoggingModule.DB);
  });

  db.on('error', (error) => {
    logModuleError(`Database error event: ${error}`, LoggingModule.DB);
  });
};

export default hookEvents;
