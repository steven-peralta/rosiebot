import WinstonCloudwatch from 'winston-cloudwatch';
import logger, { logModuleInfo } from '$util/logger';
import dbEvents from '$db/dbEvents';
import db from '$db/db';
import discordEvents from '$discord/discordEvents';
import client from '$discord/client';

const version = process.env.npm_package_version;
const nodeEnv = process.env.NODE_ENV ?? 'development';

if (nodeEnv && nodeEnv !== 'development') {
  logger().add(
    new WinstonCloudwatch({
      logGroupName: `rosiebot-${nodeEnv}`,
      logStreamName: `rosiebot-${nodeEnv}-info`,
      awsRegion: 'us-east-1',
    })
  );
  logger().add(
    new WinstonCloudwatch({
      level: 'error',
      logGroupName: `rosiebot-${nodeEnv}`,
      logStreamName: `rosiebot-${nodeEnv}-error`,
      awsRegion: 'us-east-1',
    })
  );
}

const startBot = () => {
  logModuleInfo(`Started Rosiebot ${version}. Env: ${nodeEnv}`);
  dbEvents(db);
  discordEvents(client);
};

startBot();
