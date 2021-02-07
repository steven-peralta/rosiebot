import * as crypto from 'crypto';
import {
  CommandBuilder,
  CommandCallback,
  CommandFormatter,
  CommandMetadata,
  CommandProcessor,
  UserParams,
} from '$commands/types';
import { Command, ErrorMessage, StatusCode } from '$util/enums';
import APIField from '$util/APIField';
import config from '$config';
import { userModel } from '$db/models/User';

const fullDay = 86400; // in seconds

const metadata: CommandMetadata = {
  name: Command.wdaily,
  description: 'Get your daily dose of waifu coins',
  arguments: 'none',
  supportsDM: false,
};

export interface WDailyResponse {
  coins?: number;
  criticalRoll?: boolean;
  lastClaimed?: Date;
}

const command: CommandCallback<WDailyResponse, UserParams> = async (params) => {
  if (params) {
    const { sender, guild } = params;
    const user = await userModel.findOneOrCreate(sender, guild);
    if (user) {
      const { [APIField.dailyLastClaimed]: lastClaimed } = user;
      if ((Date.now() - lastClaimed.getTime()) / 1000 > fullDay) {
        const roll = crypto.randomInt(1, 100);
        let coins = config.daily;
        let criticalRoll = false;
        if (roll === 1) {
          coins *= 5;
          criticalRoll = true;
        } else if (roll > 1 && roll <= 21) {
          coins *= 2;
          criticalRoll = true;
        }
        user[APIField.coins] += coins;
        user.setDailyClaimed();
        await user.save();
        return {
          statusCode: StatusCode.Success,
          data: {
            coins,
            criticalRoll,
          },
        };
      }
      return {
        statusCode: StatusCode.Success,
        data: { lastClaimed },
      };
    }
  }
  return {
    statusCode: StatusCode.Error,
  };
};

const processor: CommandProcessor<WDailyResponse> = async (msg) =>
  msg.guild
    ? command({ sender: msg.author, guild: msg.guild })
    : { statusCode: StatusCode.Error };

const formatter: CommandFormatter<WDailyResponse, never> = (result, user) => {
  const { statusCode, data } = result;
  if (data) {
    const { lastClaimed, coins, criticalRoll } = data;
    if (lastClaimed) {
      const timeSince = Math.floor((Date.now() - lastClaimed.getTime()) / 1000);

      const hoursRemaining = Math.floor((fullDay - timeSince) / 60 / 60);
      const minutesRemaining = Math.floor(((fullDay - timeSince) / 60) % 60);
      const secondsRemaining = Math.floor((fullDay - timeSince) % 60);

      const seconds =
        secondsRemaining >= 10 ? `${secondsRemaining}` : `0${secondsRemaining}`;
      const minutes =
        minutesRemaining >= 10 ? `${minutesRemaining}` : `0${minutesRemaining}`;
      const hours =
        hoursRemaining >= 10 ? `${hoursRemaining}` : `0${hoursRemaining}`;

      return {
        content: `${user} You've already claimed your daily for today. You can claim again in ${hours}:${minutes}:${seconds}`,
      };
    }
    if (coins && criticalRoll) {
      return {
        content: `${user} :sparkles: **CRITICAL ROLL!!** :sparkles: You claimed :coin: ${coins} coins!`,
      };
    }
    return {
      content: `${user} You claimed :coin: ${coins} coins`,
    };
  }
  return { content: ErrorMessage[statusCode] };
};

const wdaily: CommandBuilder<WDailyResponse, UserParams, never> = {
  metadata,
  processor,
  formatter,
  command,
};

export default wdaily;
