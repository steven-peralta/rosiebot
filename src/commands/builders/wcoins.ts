import { User as DiscordUser } from 'discord.js';
import {
  CommandBuilder,
  CommandCallback,
  CommandFormatter,
  CommandMetadata,
  CommandProcessor,
  CommandResult,
  TargetedUserParams,
} from '$commands/types';
import { Command, ErrorMessage, StatusCode } from '$util/enums';
import APIField from '$util/APIField';
import { logCommandException } from '$commands/logging';
import processTargetedCommand from '$commands/processors';
import { userModel } from '$db/models/User';

export interface WCoinsResponse {
  coins: number;
  sender: DiscordUser;
  target?: DiscordUser;
}

const metadata: CommandMetadata = {
  name: Command.wcoins,
  description: 'Used to see how many coins you or another user has',
  arguments: '[@user]',
  supportsDM: false,
};

const command: CommandCallback<WCoinsResponse, TargetedUserParams> = async (
  params
): Promise<CommandResult<WCoinsResponse>> => {
  if (params) {
    const { sender, guild, target } = params;

    try {
      const user = target
        ? await userModel.findOneOrCreate(target, guild)
        : await userModel.findOneOrCreate(sender, guild);
      if (user) {
        return {
          statusCode: StatusCode.Success,
          data: {
            coins: user[APIField.coins],
            sender,
            target,
          },
        };
      }
      return { statusCode: StatusCode.Error };
    } catch (e) {
      const error = e as Error;
      logCommandException(error, metadata);
      return {
        statusCode: StatusCode.Error,
      };
    }
  }
  return {
    statusCode: StatusCode.Error,
  };
};

const processor: CommandProcessor<WCoinsResponse> = async (msg, args) =>
  processTargetedCommand(msg, args, command);

const formatter: CommandFormatter<WCoinsResponse, never> = (result, user) => {
  const { statusCode, data } = result;
  if (statusCode === StatusCode.Success) {
    if (data) {
      const { coins, target } = data;
      if (target) {
        return {
          content: `${user}: ${target} has :coin: ${
            coins === 1 ? `1 coin` : `${coins} coins`
          }`,
        };
      }
      return {
        content: `${user} You have :coin: ${
          coins === 1 ? `1 coin` : `${coins} coins`
        }`,
      };
    }
  }
  return { content: ErrorMessage[statusCode] };
};

const wcoins: CommandBuilder<WCoinsResponse, TargetedUserParams, never> = {
  metadata,
  processor,
  formatter,
  command,
};

export default wcoins;
