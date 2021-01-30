// import 'module-alias/register';
import { logModuleInfo } from '@util/logger';
import dbEvents from '@db/dbEvents';
import db from '@db/db';
import discordEvents from '@discord/discordEvents';
import client from '@discord/client';

const version = process.env.npm_package_version;

const startBot = () => {
  logModuleInfo(`Started Rosiebot ${version}`);
  dbEvents(db);
  discordEvents(client);
};

startBot();
