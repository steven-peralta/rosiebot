import { Client } from 'discord.js';
import { logInfo } from '../util/logger';

const hookEvents = (client: Client): void => {
  client.on('message', (message) => {
    logInfo(`${message.author} said ${message.content}`, 'discord');
  });
};

export default hookEvents;
