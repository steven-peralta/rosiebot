import Mongoose from 'mongoose';
import WinstonCloudwatch from 'winston-cloudwatch';
import config from '$config';
import { waifuModel } from '$db/models/Waifu';
import APIField from '$util/APIField';
import logger, { initLogger, setLogger } from '$util/logger';

const nodeEnv = process.env.NODE_ENV ?? 'development';

const { mongodbUri } = config;
const db = Mongoose.connection;

const upToId: number = process.argv[2] ? parseInt(process.argv[2], 10) : 30000;

setLogger(initLogger('scrape'));

if (nodeEnv && nodeEnv !== 'development') {
  logger().add(
    new WinstonCloudwatch({
      logGroupName: `rosiebot-${nodeEnv}-scrape`,
      logStreamName: `rosiebot-${nodeEnv}-scrape-info`,
      awsRegion: 'us-east-1',
    })
  );
  logger().add(
    new WinstonCloudwatch({
      level: 'error',
      logGroupName: `rosiebot-${nodeEnv}-scrape`,
      logStreamName: `rosiebot-${nodeEnv}-scrape-error`,
      awsRegion: 'us-east-1',
    })
  );
}

Mongoose.connect(mongodbUri, {
  useNewUrlParser: true,
  useFindAndModify: true,
  useUnifiedTopology: true,
  useCreateIndex: true,
}).catch(logger().error);

const cacheWaifu = async (id: number) => {
  const waifu = await waifuModel.updateFromMWL(id);
  if (waifu) {
    logger().info(
      `${id}/${upToId} (${Math.round((id / upToId) * 100)}%): ${
        waifu[APIField.name]
      }`
    );
  }
};

const scrape = async () => {
  for (let i = 1; i < upToId; i += 1) {
    try {
      // eslint-disable-next-line no-await-in-loop
      await cacheWaifu(i);
    } catch (e) {
      if (e.message) {
        logger().error(
          `${i}/${upToId} (${Math.round((i / upToId) * 100)}%): ${e.message}`
        );
      }
      if (e.response && e.response.status === 429) {
        logger().error('Rate limited, trying again...');
        i -= 1;
      }
    }
  }
  await waifuModel.updateScoresAndTiers();
};

db.once('open', () => {
  logger().info('Connected to database');
  scrape().then(() => {
    logger().info('finished');
    process.exit(0);
  });
});

db.on('error', () => {
  logger().error('Error connecting to database');
});
