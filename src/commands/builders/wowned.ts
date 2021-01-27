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
import processTargetedCommand from '@commands/processors';
import formatWaifuResults from '@commands/formatters';
import { User as DiscordUser } from 'discord.js';

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

const command: CommandCallback<WOwnedResponse, TargetedUserParams> = async (
  params
): Promise<CommandResult<WOwnedResponse>> => {
  try {
    const { sender, guild, target } = params as TargetedUserParams;
    const start = Date.now();
    const user = target
      ? await userModel.findOneOrCreate(target, guild)
      : await userModel.findOneOrCreate(sender, guild);

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
    result.data?.userDoc,
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
