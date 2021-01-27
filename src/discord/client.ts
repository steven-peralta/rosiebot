import { Client } from 'discord.js';
import config from 'rosiebot/src/config';
import { LoggingModule, logModuleError } from 'rosiebot/src/util/logger';

const initDiscord = () => {
  const discordClient = new Client();
  discordClient
    .login(config.discordTokenKey)
    .catch((err) => logModuleError(err, LoggingModule.Discord));
  return discordClient;
};

const discordClientInstance = initDiscord();

export default discordClientInstance;
