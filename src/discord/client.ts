import { Client } from 'discord.js';
import config from '@config';
import { LoggingModule, logModuleError } from '@util/logger';

const initDiscord = () => {
  const discordClient = new Client();
  discordClient
    .login(config.discordTokenKey)
    .catch((err) => logModuleError(err, LoggingModule.Discord));
  return discordClient;
};

const client = initDiscord();

export default client;
