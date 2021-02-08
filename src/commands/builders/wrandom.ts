import {
  CommandBuilder,
  CommandCallback,
  CommandFormatter,
  CommandMetadata,
  CommandProcessor,
  CommandResult,
} from '$commands/types';
import { Command, StatusCode } from '$util/enums';
import Waifu, { waifuModel } from '$db/models/Waifu';
import { logCommandException } from '$commands/logging';
import formatWaifuResults from '$commands/formatters';

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
    const doc = await waifuModel.random();
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

const wrandom: CommandBuilder<Waifu, undefined, Waifu> = {
  metadata,
  processor,
  formatter,
  command,
};

export default wrandom;
