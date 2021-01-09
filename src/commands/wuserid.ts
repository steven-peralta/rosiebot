import { APIMessage, Guild, GuildMember, User } from 'discord.js';
import {
  Command,
  CommandBuilder,
  CommandFormatter,
  CommandMetadata,
  CommandProcessor,
} from './types';
import CommandName from './CommandName';
import { getId, getUsersFromMentionsStr } from '../discord/utils';
import Result, { StatusCode } from './Result';
import { logError } from '../util/logger';

const metadata: CommandMetadata = {
  name: CommandName.wuserid,
  description: 'Prints the user id',
  arguments: '[@user]',
  supportsDM: false,
};

const command: Command = async (
  sender: User,
  guild: Guild,
  target?: GuildMember
): Promise<Result> => {
  const userId = target
    ? getId(target.id, guild.id)
    : getId(sender.id, guild.id);

  return {
    statusCode: StatusCode.Completed,
    result: userId,
  };
};

const processor: CommandProcessor = async (msg, args) => {
  if (msg.guild) {
    const { author } = msg;
    let target;
    if (args.length > 0) {
      target = getUsersFromMentionsStr(msg.guild, [args[0]]);
    }
    if (target) {
      return command(author, msg.guild, target);
    }
    return command(author, msg.guild);
  }
  logError(`${metadata.name}: message didn't have a valid guild`);
  return {
    statusCode: StatusCode.Error,
  };
};

const formatter: CommandFormatter = (channel, result) => {
  return new APIMessage(channel, { content: result.result });
};

const builder: CommandBuilder = {
  metadata,
  processor,
  formatter,
  command,
};

export default builder;
