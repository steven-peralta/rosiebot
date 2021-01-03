import {
  Client,
  DMChannel,
  Message,
  NewsChannel,
  TextChannel,
} from 'discord.js';
import config from '../config';
import commands from '../commands/commands';
import { logInfo } from '../util/logger';

const parseCommand = (msg: Message) => {
  const { content, channel, author } = msg;

  // match our first character of the message to our command prefix
  if (content[0] === config.commandPrefix) {
    const args = content.substr(1).split(' ');
    const commandBuilder = commands[args[0]];

    if (commandBuilder) {
      const { processor, formatter, metadata } = commandBuilder;

      const logBuilder = `${author.tag} issued command ${config.commandPrefix}${metadata.name}`;
      if (
        msg.channel instanceof TextChannel ||
        msg.channel instanceof NewsChannel
      ) {
        logInfo(`${logBuilder} in the server ${msg.guild?.name}`);
      } else {
        logInfo(`${logBuilder} in the DMs of the bot`);
      }

      if (!metadata.supportsDM && channel instanceof DMChannel) {
        channel
          .send(
            `The ${metadata.name} cannot be invoked from the direct messages of the bot.`
          )
          .catch();
        return;
      }

      if (metadata.botOwnerOnly && author.id !== config.botOwner) {
        channel.send('You cannot run this command.');
        return;
      }

      channel.send('loading...').then((resultMessage) => {
        processor(msg, args).then((result) => {
          const formattedResult = formatter(channel, result);
          resultMessage.edit(formattedResult).catch();
        });
      });
    }
  }
};

const hookEvents = (client: Client): void => {
  client.on('message', (message) => {
    parseCommand(message);
  });
};

export default hookEvents;
