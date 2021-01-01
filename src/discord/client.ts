import { Client } from 'discord.js';
import config from '../config';
import { logError, logInfo } from '../util/logger';

const initDiscord = () => {
  const discordClient = new Client();
  discordClient
    .login(config.discordTokenKey)
    .then(() => logInfo('Logged in to Discord', 'discord'))
    .catch((err) => logError(err, 'discord'));
  return discordClient;
};

const client = initDiscord();

export default client;
