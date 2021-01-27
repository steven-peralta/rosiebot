import Waifu, { waifuModel } from '@db/models/Waifu';
import User, { userModel } from '@db/models/User';
import {
  CommandBuilder,
  CommandCallback,
  CommandFormatter,
  CommandMetadata,
  CommandProcessor,
  CommandResult,
  UserParams,
} from '@commands/types';
import { Command, ErrorMessage, StatusCode } from '@util/enums';
import APIField from '@util/APIField';
import config from '@config';
import { getWotd } from '@commands/builders/wotd';
import { logCommandException } from '@commands/logging';
import { formatWaifuResults } from '@commands/formatters';
import randomOrg from '@api/random-org/randomOrg';

export interface WRollResponse {
  waifu: Waifu;
  criticalRoll: boolean;
  wotdRoll: boolean;
  userDoc: User;
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
      const user = await userModel.findOneOrCreate(sender, guild);

      if (user) {
        if (user[APIField.coins] >= config.rollCost) {
          const start = Date.now();
          const roll = await randomOrg.generateInteger(1, 100);
          let criticalRoll = false;
          let wotdRoll = false;
          let waifu: Waifu | undefined;

          if (roll === 1) {
            // wotd roll
            waifu = (await getWotd())?.waifu;
            wotdRoll = true;
          } else if (roll >= 2 && roll <= 21) {
            // ranked waifu roll
            waifu = await waifuModel.getRandom({
              [APIField.tier]: { $ne: undefined },
            });
            criticalRoll = true;
          } else {
            // regular roll
            waifu = await waifuModel.getRandom();
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
                userDoc: user,
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
      const { waifu, criticalRoll, wotdRoll, userDoc } = data;
      const criticalRollContent = ` :sparkles: **CRITICAL ROLL!!** :sparkles:`;
      const wotdRollContent = ` :star2: **You rolled the Waifu of the Day. Congrats!**`;

      return formatWaifuResults(
        statusCode,
        `${user}${criticalRoll ? criticalRollContent : ''}${
          wotdRoll ? wotdRollContent : ''
        } Here's who you rolled:\n`,
        waifu,
        user,
        userDoc,
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
