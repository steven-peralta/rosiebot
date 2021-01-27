import WaifuModel, { Waifu } from 'rosiebot/src/db/models/Waifu';
import {
  CommandBuilder,
  CommandFormatter,
  CommandCallback,
  CommandMetadata,
  CommandProcessor,
  CommandResult,
} from 'rosiebot/src/commands/types';
import randomOrgAPI from 'rosiebot/src/api/random-org/RandomOrgAPI';
import { Command, StatusCode } from 'rosiebot/src/util/enums';
import APIField from 'rosiebot/src/util/APIField';
import { formatWaifuResults } from 'rosiebot/src/commands/formatters';
import { logCommandException } from 'rosiebot/src/commands/logging';

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
    const random = await randomOrgAPI.generateInteger(1, 5);
    const waifu = await WaifuModel.getRandom({ [APIField.tier]: random });
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
    logCommandException(e, metadata);
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
