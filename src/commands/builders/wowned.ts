import Waifu from '@db/models/Waifu';
import User, { userModel } from '@db/models/User';
import {
  CommandBuilder,
  CommandCallback,
  CommandFormatter,
  CommandMetadata,
  CommandProcessor,
  CommandResult,
  TargetedUserParams,
} from '@commands/types';
import { Command, StatusCode } from '@util/enums';
import APIField from '@util/APIField';
import { isDocumentArray } from '@typegoose/typegoose';
import { logCommandException } from '@commands/logging';
import formatWaifuResults from '@commands/formatters';
import { User as DiscordUser } from 'discord.js';
import { getUsersFromMentionsStr } from '@util/string';
import parseWaifuSearchArgs from '@util/args';

interface WOwnedResponse {
  ownedWaifus: Waifu[];
  userDoc?: User;
  target?: DiscordUser;
}

const metadata: CommandMetadata = {
  name: Command.wowned,
  description: 'Used to see the waifus that you or another user owns',
  arguments: '[@user]',
  supportsDM: false,
};

const command: CommandCallback<
  WOwnedResponse,
  TargetedUserParams & { args?: string[] }
> = async (params): Promise<CommandResult<WOwnedResponse>> => {
  try {
    const { sender, guild, target, args } = params as TargetedUserParams & {
      args: string[];
    };
    const start = Date.now();
    const user = target
      ? await userModel.findOneOrCreate(target, guild)
      : await userModel.findOneOrCreate(sender, guild);

    if (user) {
      let options = {};
      if (args && args.length > 0) {
        const searchArgs = parseWaifuSearchArgs(args);
        options = {
          conditions: searchArgs.conditions,
          sort: searchArgs.sort,
          projections: searchArgs.projection,
        };
      }
      await user
        .populate({
          path: APIField.ownedWaifus,
          populate: [{ path: APIField.appearances }, { path: APIField.series }],
          options,
        })
        .execPopulate();
      const { [APIField.ownedWaifus]: ownedWaifus } = user;
      if (isDocumentArray(ownedWaifus)) {
        if (ownedWaifus.length === 0) {
          return {
            statusCode: StatusCode.UserOwnsNoWaifus,
          };
        }
        return {
          statusCode: StatusCode.Success,
          data: {
            ownedWaifus,
            userDoc: target
              ? undefined
              : await userModel.findOneOrCreate(sender, guild),
          },
          time: Date.now() - start,
        };
      }
    }

    return { statusCode: StatusCode.Error };
  } catch (e) {
    logCommandException(e, metadata);
    return { statusCode: StatusCode.Error };
  }
};

const processor: CommandProcessor<WOwnedResponse> = async (msg, args) => {
  const { guild, author: sender } = msg;
  if (guild) {
    if (args && args.length > 0) {
      const [targetStr, ...rest] = args;

      if (args.length === 1 && targetStr.match(':')) {
        // check if this is a special search param...
        return command({
          sender,
          guild,
          args,
        });
      }

      const [target] = getUsersFromMentionsStr(msg.guild, [targetStr]) ?? [];
      if (target) {
        return command({
          sender: msg.author,
          target,
          guild,
          args: rest,
        });
      }
      return {
        statusCode: StatusCode.UserNotFound,
      };
    }
    return command({ guild, sender });
  }
  return {
    statusCode: StatusCode.Error,
  };
};

const formatter: CommandFormatter<WOwnedResponse, Waifu> = (
  result,
  user: DiscordUser
) =>
  formatWaifuResults(
    result.statusCode,
    `${user}`,
    result.data?.ownedWaifus,
    user,
    result.data?.userDoc,
    user ? [user] : [],
    result.time
  );

const wowned: CommandBuilder<
  WOwnedResponse,
  TargetedUserParams & { args: string[] },
  Waifu
> = {
  metadata,
  processor,
  formatter,
  command,
};

export default wowned;
