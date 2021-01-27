import Mongoose from 'mongoose';
import { LoggingModule, logModuleError } from 'rosiebot/src/util/logger';
import config from 'rosiebot/src/config';

const initDb = () => {
  Mongoose.connect(config.mongodbUri, {
    useNewUrlParser: true,
    useFindAndModify: true,
    useUnifiedTopology: true,
    useCreateIndex: true,
    keepAlive: true,
    keepAliveInitialDelay: 300000,
  }).catch((e) =>
    logModuleError(
      `Exception caught while connecting to the database: ${e}`,
      LoggingModule.DB
    )
  );

  return Mongoose.connection;
};

const dbInstance = initDb();

export default dbInstance;
