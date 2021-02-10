import Mongoose from 'mongoose';
import { waifuModel } from '$db/models/Waifu';
import config from '$config';
import logger, { initLogger, setLogger } from '$util/logger';

const { mongodbUri } = config;
const db = Mongoose.connection;

setLogger(initLogger('rank'));

Mongoose.connect(mongodbUri, {
  useNewUrlParser: true,
  useFindAndModify: true,
  useUnifiedTopology: true,
  useCreateIndex: true,
}).catch(logger().error);

const rank = async () => {
  try {
    await waifuModel.updateScoresAndTiers();
  } catch (e) {
    logger().error(e);
  }
};

db.on('open', () => {
  logger().info('Starting ranking process...');
  rank()
    .then(() => {
      logger().info('done');
    })
    .catch((e) => {
      logger().error(e);
    });
});
