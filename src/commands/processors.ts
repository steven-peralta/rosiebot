import { Message } from 'discord.js';
import {
  CommandCallback,
  CommandResult,
  TargetedUserParams,
} from '$commands/types';
import { getUsersFromMentionsStr } from '$util/string';
import { StatusCode } from '$util/enums';

const processTargetedCommand = async <ResultType>(
  msg: Message,
  args: string[],
  command: CommandCallback<ResultType, TargetedUserParams>
): Promise<CommandResult<ResultType>> => {
  const { author: sender, guild } = msg;
  if (guild) {
    let target;
    if (args.length > 0) {
      target = getUsersFromMentionsStr(guild, [args[0]]);
    }
    if (target && target.length > 0) {
      return command({
        sender,
        guild,
        target: target[0],
      });
    }
    return command({ sender, guild });
  }
  return {
    statusCode: StatusCode.Error,
  };
};

export default processTargetedCommand;
