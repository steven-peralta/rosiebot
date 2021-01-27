import { logModuleInfo } from 'rosiebot/src/util/logger';
import dbInstance from 'rosiebot/src/db/db';
import hookDBEvents from 'rosiebot/src/db/events';
import hookDiscordEvents from 'rosiebot/src/discord/events';
import discordClientInstance from 'rosiebot/src/discord/client';

const version = process.env.npm_package_version;

const startBot = () => {
  logModuleInfo(`Started Rosiebot ${version}`);
  hookDBEvents(dbInstance);
  hookDiscordEvents(discordClientInstance);
};

startBot();
