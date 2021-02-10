import Mongoose from 'mongoose';

import { Client, Snowflake } from 'discord.js';
import { userModel } from '$db/models/User';
import { waifuModel } from '$db/models/Waifu';
import config from '$config';

const { mongodbUri } = config;
const db = Mongoose.connection;

const discordClient = new Client();

Mongoose.connect(mongodbUri, {
  useNewUrlParser: true,
  useFindAndModify: true,
  useUnifiedTopology: true,
  useCreateIndex: true,
}).catch(console.error);

async function start() {
  const oldUserColl = db.useDb('old').collection('users');
  const oldUsers = await oldUserColl.find({}).toArray();

  for (const oldUser of oldUsers) {
    let { coins } = oldUser;
    const { userId, ownedWaifus } = oldUser;
    let userSnowflake: Snowflake;
    let serverSnowflake: Snowflake;

    if (userId.length === 35) {
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
      console.error(`Unable to fetch user: ${e}`);
      continue;
    }
    let discordServer;
    try {
      discordServer = await discordClient.guilds.fetch(serverSnowflake);
    } catch (e) {
      console.error(`Unable to fetch guild: ${e}`);
      continue;
    }

    const newWaifuList = [];
    for (let i = 0; i < ownedWaifus.length; i += 1) {
      const waifu = await waifuModel.findById(ownedWaifus[i]).lean();
      if (!waifu) {
        coins += 200;
      } else {
        newWaifuList.push(waifu._id);
      }
    }

    const newUserData = {
      user_id: userSnowflake,
      discord_tag: discordUser.tag,
      server_id: serverSnowflake,
      server_name: discordServer.name,
      coins,
      created: oldUser.dateOfEntry,
      updated: new Date(),
      daily_last_claimed: oldUser.dailyLastClaimed,
      owned_waifus: newWaifuList,
    };

    await userModel.create(newUserData);
  }

  console.log('done');
}

console.log('Logging into discord');
discordClient
  .login(config.discordTokenKey)
  .then(() => {
    console.log('Logged in.');
    start();
  })
  .catch((err) => {
    console.error(err);
  });
