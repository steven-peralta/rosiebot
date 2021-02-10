import Mongoose from 'mongoose';

import { Client, Snowflake } from 'discord.js';
import { DocumentType } from '@typegoose/typegoose';
import User, { userModel } from '$db/models/User';
import Waifu, { waifuModel } from '$db/models/Waifu';
import config from '$config';
import logger, { initLogger, setLogger } from '$util/logger';
import APIField from '$util/APIField';

const { mongodbUri } = config;
const db = Mongoose.connection;

const discordClient = new Client();

setLogger(initLogger('rank'));

Mongoose.connect(mongodbUri, {
  useNewUrlParser: true,
  useFindAndModify: true,
  useUnifiedTopology: true,
  useCreateIndex: true,
}).catch(logger().error);

const processWaifu = async (oldWaifuId: number) => {
  const waifu = await waifuModel.findById(oldWaifuId).lean();
  if (!waifu) return undefined;
  return waifu;
};

const processWaifus = async (oldWaifuList: number[]) => {
  const promises: Promise<DocumentType<Waifu> | undefined>[] = [];
  oldWaifuList.forEach((oldWaifuId) => {
    promises.push(processWaifu(oldWaifuId));
  });
  return Promise.all(promises);
};

const processUser = async (
  oldUser: Record<string, unknown>
): Promise<DocumentType<User> | undefined> => {
  let coins: number = oldUser.coins as number;
  const userId: string = oldUser.userId as string;
  const ownedWaifus: number[] = oldUser.ownedWaifus as number[];

  let userSnowflake: Snowflake;
  let serverSnowflake: Snowflake;
  if ((userId as string).length === 35) {
    userSnowflake = userId.substring(0, 17);
    serverSnowflake = userId.substring(17);
  } else if (userId.length === 36) {
    userSnowflake = userId.substring(0, 18);
    serverSnowflake = userId.substring(18);
  } else {
    throw new Error('userId was an unexpected length!');
  }

  let discordUser;
  try {
    discordUser = await discordClient.users.fetch(userSnowflake);
  } catch (e) {
    logger().error(`Unable to fetch user: ${e}`);
    return undefined;
  }
  let discordServer;
  try {
    discordServer = await discordClient.guilds.fetch(serverSnowflake);
  } catch (e) {
    logger().error(`Unable to fetch guild: ${e}`);
    return undefined;
  }

  const processedWaifus = await processWaifus(ownedWaifus);
  processedWaifus.forEach((waifu) => {
    if (!waifu) {
      coins += 200;
    }
  });

  const filteredWaifus: number[] = processedWaifus.reduce<number[]>(
    (acc, val) => {
      const arr = acc;
      if (val !== undefined) {
        arr.push(val[APIField._id]);
      }
      return arr;
    },
    [] as number[]
  );

  coins += 2000;
  const newUserData = {
    [APIField.userId]: userSnowflake,
    [APIField.serverId]: serverSnowflake,
    [APIField.discordTag]: discordUser.tag,
    [APIField.serverName]: discordServer.name,
    [APIField.coins]: coins,
    [APIField.created]: oldUser.dateOfEntry as Date,
    [APIField.updated]: new Date(),
    [APIField.dailyLastClaimed]: oldUser.dailyLastClaimed as Date,
    [APIField.ownedWaifus]: filteredWaifus,
  };

  return userModel.create(newUserData);
};

const start = async () => {
  const oldUserColl = db.useDb('old').collection('users');
  const oldUsers = await oldUserColl.find({}).toArray();
  const promises: Promise<DocumentType<User> | undefined>[] = [];

  oldUsers.forEach((oldUser) => {
    promises.push(processUser(oldUser));
  });

  await Promise.all(promises);
};

logger().info('Logging into discord');
discordClient
  .login(config.discordTokenKey)
  .then(() => {
    logger().info('Logged in.');
    start().then(() => {
      logger().info('done');
      process.exit(0);
    });
  })
  .catch((err) => {
    logger().error(err);
  });
