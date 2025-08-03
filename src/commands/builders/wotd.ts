import * as crypto from 'crypto';
import Waifu, { waifuModel } from '$db/models/Waifu';
import {
  CommandBuilder,
  CommandCallback,
  CommandFormatter,
  CommandMetadata,
  CommandProcessor,
  CommandResult,
} from '$commands/types';
import { Command, StatusCode } from '$util/enums';
import APIField from '$util/APIField';
import { logCommandException } from '$commands/logging';
import formatWaifuResults from '$commands/formatters';

interface Wotd {
  waifu: Waifu;
  expiresOn: Date;
}

let currentWotd: Wotd;

const metadata: CommandMetadata = {
  name: Command.wotd,
  description: 'Shows the waifu of the day',
  supportsDM: true,
};

export const getWotd = async (): Promise<Wotd | undefined> => {
  if (!currentWotd || Date.now() >= currentWotd.expiresOn.getTime()) {
    const random = crypto.randomInt(1, 5);
    const waifu = await waifuModel.random({ [APIField.tier]: random });
    const expiresOn = new Date();
    expiresOn.setHours(24, 0, 0, 0);
    if (waifu) {
      currentWotd = {
        waifu,
        expiresOn,
      };
      return currentWotd;
    }
  }
  return currentWotd;
};

const command: CommandCallback<Waifu, undefined> = async (): Promise<
  CommandResult<Waifu>
> => {
  try {
    const time = Date.now();
    const wotd = await getWotd();
    if (wotd) {
      return {
        statusCode: StatusCode.Success,
        data: wotd.waifu,
        time: Date.now() - time,
      };
    }
    return {
      statusCode: StatusCode.Error,
    };
  } catch (e) {
    const error = e as Error;
    logCommandException(error, metadata);
    return {
      statusCode: StatusCode.Error,
    };
  }
};

const processor: CommandProcessor<Waifu> = async (_msg, _args) => command();

const formatter: CommandFormatter<Waifu, Waifu> = (result, user) => {
  const date = new Date();
  const secondsRemaining = 60 - date.getSeconds();
  const minutesRemaining = 59 - date.getMinutes();
  const hoursRemaining = 23 - date.getHours();

  const seconds =
    secondsRemaining >= 10 ? `${secondsRemaining}` : `0${secondsRemaining}`;
  const minutes =
    minutesRemaining >= 10 ? `${minutesRemaining}` : `0${minutesRemaining}`;
  const hours =
    hoursRemaining >= 10 ? `${hoursRemaining}` : `0${hoursRemaining}`;

  return formatWaifuResults(
    result.statusCode,
    `${user} Here's the Waifu of the Day:\nRefreshes in ${hours}:${minutes}:${seconds}`,
    result.data,
    user,
    undefined,
    undefined,
    result.time
  );
};

const wotd: CommandBuilder<Waifu, undefined, Waifu> = {
  metadata,
  processor,
  formatter,
  command,
};

export default wotd;
