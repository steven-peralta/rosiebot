import { Client as DiscordClient } from 'discord.js';

import config from './config';
import commands from './discord/commands';
import Mongoose from 'mongoose';
const { discordTokenKey, commandPrefix, mongo } = config;
const discordClient = new DiscordClient();
const db = Mongoose.connection;

Mongoose.connect(`mongodb://${mongo.hostname}:${mongo.port}/${mongo.db}`, {
  useNewUrlParser: true,
  useFindAndModify: true,
  useUnifiedTopology: true,
  useCreateIndex: true,
}).catch(console.error);

db.once('open', async () => {
  console.log('Connected to database');
});

db.on('error', () => {
  console.log('Error connecting to database');
});

discordClient.on('message', async (msg) => {
  const { content } = msg;

  if (content[0] === commandPrefix) {
    const commandArgs = content.substr(1).split(' ');
    const command = commands[commandArgs[0]];
    if (command) await command(msg, ...commandArgs).catch(console.error);
  }
});
discordClient.login(discordTokenKey).catch(console.error);
