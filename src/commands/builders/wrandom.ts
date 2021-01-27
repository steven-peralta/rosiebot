import {
  CommandBuilder,
  CommandFormatter,
  CommandCallback,
  CommandMetadata,
  CommandProcessor,
  CommandResult,
} from 'rosiebot/src/commands/types';
import { Command, StatusCode } from 'rosiebot/src/util/enums';
import WaifuModel, { Waifu } from 'rosiebot/src/db/models/Waifu';
import { formatWaifuResults } from 'rosiebot/src/commands/formatters';
import { logCommandException } from 'rosiebot/src/commands/logging';

const metadata: CommandMetadata = {
  name: Command.wrandom,
  description: 'Pull a random-org waifu',
  supportsDM: true,
};

const command: CommandCallback<Waifu, undefined> = async (): Promise<
  CommandResult<Waifu>
> => {
  try {
    const start = Date.now();
    const doc = await WaifuModel.getRandom();
    if (doc) {
      return {
        data: doc,
        statusCode: StatusCode.Success,
        time: Date.now() - start,
      };
    }
    return {
      statusCode: StatusCode.Error,
    };
  } catch (e) {
    logCommandException(e, metadata);
    return { statusCode: StatusCode.Error };
  }
};

const processor: CommandProcessor<Waifu> = async (_msg, _args) => command();

const formatter: CommandFormatter<Waifu, Waifu> = (result, user) =>
  formatWaifuResults(
    result.statusCode,
    '',
    result.data,
    user,
    undefined,
    [user],
    result.time
  );

const builder: CommandBuilder<Waifu, undefined, Waifu> = {
  metadata,
  processor,
  formatter,
  command,
};

export default builder;
