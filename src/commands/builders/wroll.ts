import {
  CommandBuilder,
  CommandFormatter,
  CommandCallback,
  CommandMetadata,
  CommandProcessor,
  CommandResult,
  UserParams,
} from 'rosiebot/src/commands/types';
import { Command, StatusCode, ErrorMessage } from 'rosiebot/src/util/enums';
import WaifuModel, { Waifu } from 'rosiebot/src/db/models/Waifu';
import randomOrgAPIInstance from 'rosiebot/src/api/random-org/RandomOrgAPI';
import APIField from 'rosiebot/src/util/APIField';
import UserModel, { User } from 'rosiebot/src/db/models/User';
import { logCommandException } from 'rosiebot/src/commands/logging';
import config from 'rosiebot/src/config';
import { getWotd } from 'rosiebot/src/commands/builders/wotd';
import { formatWaifuResults } from 'rosiebot/src/commands/formatters';

export interface WRollResponse {
  waifu: Waifu;
  criticalRoll: boolean;
  wotdRoll: boolean;
  userModel: User;
}

const metadata: CommandMetadata = {
  name: Command.wroll,
  description: 'Roll for a waifu',
  supportsDM: false,
};

const command: CommandCallback<WRollResponse, UserParams> = async (
  params
): Promise<CommandResult<WRollResponse>> => {
  if (params) {
    const { sender, guild }: UserParams = params;

    try {
      const user = await UserModel.findOneOrCreate(sender, guild);

      if (user) {
        if (user[APIField.coins] >= config.rollCost) {
          const start = Date.now();
          const roll = await randomOrgAPIInstance.generateInteger(1, 100);
          let criticalRoll = false;
          let wotdRoll = false;
          let waifu: Waifu | undefined;

          if (roll === 1) {
            // wotd roll
            waifu = (await getWotd())?.waifu;
            wotdRoll = true;
          } else if (roll >= 2 && roll <= 21) {
            // ranked waifu roll
            waifu = await WaifuModel.getRandom({
              [APIField.tier]: { $ne: undefined },
            });
            criticalRoll = true;
          } else {
            // regular roll
            waifu = await WaifuModel.getRandom();
          }

          if (waifu) {
            // if that user already owns the waifu, roll again
            if (user[APIField.ownedWaifus].includes(waifu[APIField._id])) {
              return command(params);
            }

            // subtract our coins and append our waifu to the user's owned list
            user[APIField.coins] -= config.rollCost;
            user[APIField.ownedWaifus].push(waifu[APIField._id]);
            await user.save();

            return {
              statusCode: StatusCode.Success,
              data: {
                waifu,
                criticalRoll,
                wotdRoll,
                userModel: user,
              },
              time: Date.now() - start,
            };
          }
        } else {
          return {
            statusCode: StatusCode.InsufficentCoins,
          };
        }
      }
      return {
        statusCode: StatusCode.Error,
      };
    } catch (e) {
      logCommandException(e, metadata);
      return {
        statusCode: StatusCode.Error,
      };
    }
  }
  return {
    statusCode: StatusCode.Error,
  };
};

const processor: CommandProcessor<WRollResponse> = async (msg, _args) => {
  if (msg.guild) {
    return command({ sender: msg.author, guild: msg.guild });
  }
  return {
    statusCode: StatusCode.Error,
  };
};

const formatter: CommandFormatter<WRollResponse, Waifu> = (result, user) => {
  // todo: use waifu formatter
  const { statusCode, data, time } = result;
  if (statusCode === StatusCode.Success) {
    if (data) {
      const { waifu, criticalRoll, wotdRoll, userModel } = data;
      const criticalRollContent = ` :sparkles: **CRITICAL ROLL!!** :sparkles:`;
      const wotdRollContent = ` :star2: **You rolled the Waifu of the Day. Congrats!**`;

      return formatWaifuResults(
        statusCode,
        `${user}${criticalRoll ? criticalRollContent : ''}${
          wotdRoll ? wotdRollContent : ''
        } Here's who you rolled:\n`,
        waifu,
        user,
        userModel,
        [user],
        time
      );
    }
  }
  return { content: ErrorMessage[statusCode] };
};

const wroll: CommandBuilder<WRollResponse, UserParams, Waifu> = {
  metadata,
  processor,
  formatter,
  command,
};

export default wroll;
