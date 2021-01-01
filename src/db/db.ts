import Mongoose from 'mongoose';
import config from '../config';
import { logError } from '../util/logger';

const initDb = () => {
  Mongoose.connect(config.mongodbUri, {
    useNewUrlParser: true,
    useFindAndModify: true,
    useUnifiedTopology: true,
    useCreateIndex: true,
  }).catch((err) => logError(err, 'db'));

  return Mongoose.connection;
};

const db = initDb();

export default db;
