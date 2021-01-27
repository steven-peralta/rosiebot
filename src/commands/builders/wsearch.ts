import { Message } from 'discord.js';
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
import { parseWaifuSearchArgs } from 'rosiebot/src/util/args';

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
      const data = await WaifuModel.leanWaifuQuery(
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

const builder: CommandBuilder<Waifu[], { args: string[] }, Waifu> = {
  metadata,
  processor,
  formatter,
  command,
};

export default builder;
