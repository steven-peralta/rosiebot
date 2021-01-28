import {
  CommandBuilder,
  CommandCallback,
  CommandFormatter,
  CommandMetadata,
  CommandProcessor,
  CommandResult,
} from '@commands/types';
import { Command, StatusCode } from '@util/enums';
import Waifu, { waifuModel } from '@db/models/Waifu';
import parseWaifuSearchArgs from '@util/args';
import { logCommandException } from '@commands/logging';
import { Message } from 'discord.js';
import formatWaifuResults from '@commands/formatters';

const metadata: CommandMetadata = {
  name: Command.wsearch,
  description: 'Search for a waifu',
  arguments: '<search string>',
  supportsDM: true,
};

const command: CommandCallback<Waifu[], { args: string[] }> = async (
  args
): Promise<CommandResult<Waifu[]>> => {
  if (args) {
    try {
      const start = Date.now();
      const queryOptions = parseWaifuSearchArgs(args.args);
      const data = await waifuModel.leanFind(
        queryOptions.conditions,
        queryOptions.sort,
        queryOptions.projection
      );
      if (data) {
        return {
          data,
          statusCode: StatusCode.Success,
          time: Date.now() - start,
        };
      }
      return {
        statusCode: StatusCode.NoData,
      };
    } catch (e) {
      logCommandException(e, metadata);
    }
  }

  return {
    statusCode: StatusCode.Error,
  };
};

const processor: CommandProcessor<Waifu[]> = async (msg: Message, args) => {
  return command({ args });
};

const formatter: CommandFormatter<Waifu[], Waifu> = (result, user) =>
  formatWaifuResults(
    result.statusCode,
    `${user}`,
    result.data,
    user,
    undefined,
    [user],
    result.time
  );

const wsearch: CommandBuilder<Waifu[], { args: string[] }, Waifu> = {
  metadata,
  processor,
  formatter,
  command,
};

export default wsearch;
