import {
  Client,
  DMChannel,
  Message,
  MessageEditOptions,
  MessageEmbed,
} from 'discord.js';
import config from '$config';
import commands from '$commands/commands';
import { Command } from '$util/enums';
import {
  LoggingModule,
  logModuleError,
  logModuleInfo,
  logModuleWarning,
} from '$util/logger';
import { CommandFormatter, CommandResponseType } from '$commands/types';
import Waifu from '$db/models/Waifu';
import { logCommandException, logCommandStatus } from '$commands/logging';

const parseCommand = async (msg: Message) => {
  const { content, channel, author } = msg;
  const { tag: userTag, id: userID } = author;

  // match our first character of the message to our command prefix
  if (content[0] === config.commandPrefix) {
    const args = content.substr(1).split(' ');
    const commandBuilder = commands[args[0] as Command];

    // remove the first arg since we don't need it anymore
    args.splice(0, 1);

    if (commandBuilder) {
      const { processor, metadata } = commandBuilder;

      logModuleInfo(
        `${userTag} (${userID}) issued command ${config.commandPrefix}${metadata.name}`
      );

      if (!metadata.supportsDM && channel instanceof DMChannel) {
        channel
          .send(
            `The ${metadata.name} command cannot be invoked from the direct messages of the bot.`
          )
          .catch((e) =>
            logModuleError(
              `Exception caught sending message to channel: ${e}`,
              LoggingModule.Discord
            )
          );
        return;
      }

      if (metadata.botOwnerOnly && author.id !== config.botOwner) {
        channel.send(`${author} You cannot run this command.`).catch((e) => {
          logModuleError(
            `Exception caught sending message to channel: ${e}`,
            LoggingModule.Discord
          );
        });
        return;
      }

      try {
        const loadingMessage = await channel.send('loading...');
        // if this message is still loading after 10 seconds, log this
        setTimeout(() => {
          if (loadingMessage.content === 'loading...') {
            logModuleWarning(
              `${metadata.name} is stuck in a loading state after 10 seconds`
            );
          }
        }, 10000);
        const result = await processor(msg, args);
        const formatter = commandBuilder.formatter as CommandFormatter<
          CommandResponseType<typeof result>,
          Waifu | Waifu[]
        >;
        const { statusCode } = result;
        const { embed, content: editContent } = formatter(result, author);
        const edit: MessageEditOptions = {
          content: `${editContent ? ` ${editContent}` : ''}`,
        };
        logCommandStatus(metadata, statusCode);
        if (embed) {
          if (embed instanceof MessageEmbed) {
            // regular discord embed
            loadingMessage.edit({ ...edit, embed }).catch((err) => {
              logModuleError(
                `Error editing message: ${err}`,
                LoggingModule.Discord
              );
            });
          } else {
            // built-in interactive message
            embed.setMessage(loadingMessage);
            embed.create().catch((e: Error) => {
              logModuleError(
                `Exception caught while trying to build the interactive message: ${e}`
              );
            });
          }
        } else {
          // no embed
          loadingMessage.edit({ ...edit }).catch((e) => {
            logModuleError(
              `Error editing message for final results: ${e}`,
              LoggingModule.Discord
            );
          });
        }
      } catch (e) {
        logCommandException(e, metadata);
      }
    }
  }
};

const discordEvents = (client: Client): void => {
  client.on('ready', () => {
    logModuleInfo(
      `Discord client ready. Logged in as ${client.user?.tag}`,
      LoggingModule.Discord
    );
    logModuleInfo(
      `Available to ${client.users.cache.size} users, ${client.guilds.cache.size} servers, and ${client.channels.cache.size} channels`,
      LoggingModule.Discord
    );

    client.user?.setActivity('Type !rhelp for available commands!');
    client
      .generateInvite({ permissions: ['SEND_MESSAGES', 'MANAGE_MESSAGES'] })
      .then((link) => {
        logModuleInfo(
          `Generated bot invite link: ${link}`,
          LoggingModule.Discord
        );
      });
  });

  client.on('disconnect', () => {
    logModuleError('Connection to Discord lost', LoggingModule.Discord);
  });

  client.on('reconnecting', () => {
    logModuleInfo('Reconnecting to Discord...', LoggingModule.Discord);
  });

  client.on('error', (error) => {
    logModuleInfo(`Discord error event: ${error}`, LoggingModule.Discord);
  });

  client.on('warn', (warn) => {
    logModuleWarning(`Discord warning event: ${warn}`, LoggingModule.Discord);
  });

  client.on('message', (message) => {
    parseCommand(message).catch((e) => {
      logModuleError(
        `Exception caught while parsing command: ${e}`,
        LoggingModule.Discord
      );
    });
  });
};

export default discordEvents;
