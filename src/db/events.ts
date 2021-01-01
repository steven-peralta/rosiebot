import { Connection } from 'mongoose';
import { logInfo } from '../util/logger';

const hookEvents = (db: Connection): void => {
  db.on('connecting', () => {
    logInfo('Connecting to MongoDB', 'db');
  });

  db.on('connected', () => {
    logInfo('Connected to MongoDB', 'db');
  });
};

export default hookEvents;
