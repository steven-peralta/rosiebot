import Mongoose from 'mongoose';
import config from '$config';
import { LoggingModule, logModuleError } from '$util/logger';

const initDb = () => {
  Mongoose.connect(config.mongodbUri, {
    useNewUrlParser: true,
    useFindAndModify: true,
    useUnifiedTopology: true,
    useCreateIndex: true,
    keepAlive: true,
    keepAliveInitialDelay: 300000,
    autoIndex: process.env.NODE_ENV !== 'production',
  }).catch((e) =>
    logModuleError(
      `Exception caught while connecting to the database: ${e}`,
      LoggingModule.DB
    )
  );

  return Mongoose.connection;
};

const db = initDb();

export default db;
