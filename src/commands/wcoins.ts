import { APIMessage, Guild, GuildMember, User } from 'discord.js';
import {
  Command,
  CommandBuilder,
  CommandFormatter,
  CommandMetadata,
  CommandProcessor,
} from './types';
import UserModel, { User as BotUser } from '../db/models/User';
import Result, { StatusCode } from './Result';
import { getId, getUsersFromMentionsStr } from '../discord/utils';
import ApiFields from '../util/ApiFields';
import logCommandException from './utils';
import CommandName from './CommandName';

const metadata: CommandMetadata = {
  name: CommandName.wcoins,
  description: 'Used to see how many coins you or another user has',
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

  try {
    const user: BotUser = await UserModel.findOneOrCreate(userId);
    return {
      statusCode: StatusCode.Completed,
      result: user[ApiFields.coins],
    };
  } catch (e) {
    logCommandException(e, metadata);
    return {
      statusCode: StatusCode.Error,
    };
  }
};

const processor: CommandProcessor = async (msg, args) => {
  const { author } = msg;
  let target;
  if (args) {
    target = getUsersFromMentionsStr(msg.guild!, [args[0]]);
  }
  if (target) {
    return command(author, msg.guild, target);
  }
  return command(author, msg.guild);
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
