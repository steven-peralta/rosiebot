import { isDocumentArray } from '@typegoose/typegoose';
import {
  CommandBuilder,
  CommandFormatter,
  CommandCallback,
  CommandMetadata,
  CommandProcessor,
  CommandResult,
  TargetedUserParams,
} from 'rosiebot/src/commands/types';
import { Command, StatusCode } from 'rosiebot/src/util/enums';
import { Waifu } from 'rosiebot/src/db/models/Waifu';
import UserModel, { User } from 'rosiebot/src/db/models/User';
import APIField from 'rosiebot/src/util/APIField';
import { processTargetedCommand } from 'rosiebot/src/commands/processors';
import { formatWaifuResults } from 'rosiebot/src/commands/formatters';
import { logCommandException } from 'rosiebot/src/commands/logging';
import { User as DiscordUser } from 'discord.js';

interface WOwnedResponse {
  ownedWaifus: Waifu[];
  userModel?: User;
  target?: DiscordUser;
}

const metadata: CommandMetadata = {
  name: Command.wowned,
  description: 'Used to see the waifus that you or another user owns',
  arguments: '[@user]',
  supportsDM: false,
};

const command: CommandCallback<WOwnedResponse, TargetedUserParams> = async (
  params
): Promise<CommandResult<WOwnedResponse>> => {
  try {
    const { sender, guild, target } = params as TargetedUserParams;
    const start = Date.now();
    const user = target
      ? await UserModel.findOneOrCreate(target, guild)
      : await UserModel.findOneOrCreate(sender, guild);

    if (user) {
      await user
        .populate({
          path: APIField.ownedWaifus,
          populate: [{ path: APIField.appearances }, { path: APIField.series }],
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
            userModel: target
              ? undefined
              : await UserModel.findOneOrCreate(sender, guild),
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

const processor: CommandProcessor<WOwnedResponse> = async (msg, args) =>
  processTargetedCommand(msg, args, command);

const formatter: CommandFormatter<WOwnedResponse, Waifu> = (
  result,
  user: DiscordUser
) =>
  formatWaifuResults(
    result.statusCode,
    `${user}`,
    result.data?.ownedWaifus,
    user,
    result.data?.userModel,
    user ? [user] : [],
    result.time
  );

const wowned: CommandBuilder<WOwnedResponse, TargetedUserParams, Waifu> = {
  metadata,
  processor,
  formatter,
  command,
};

export default wowned;
