import { Command } from 'rosiebot/src/util/enums';
import wcoins from 'rosiebot/src/commands/builders/wcoins';
import wsearch from 'rosiebot/src/commands/builders/wsearch';
import wowned from 'rosiebot/src/commands/builders/wowned';
import wrandom from 'rosiebot/src/commands/builders/wrandom';
import wotd from 'rosiebot/src/commands/builders/wotd';
import wroll from 'rosiebot/src/commands/builders/wroll';
import wtrade from 'rosiebot/src/commands/builders/wtrade';
import ssearch from 'rosiebot/src/commands/builders/ssearch';
import wdaily from 'rosiebot/src/commands/builders/wdaily';

const commands = {
  [Command.wcoins]: wcoins,
  [Command.wsearch]: wsearch,
  [Command.wowned]: wowned,
  [Command.wrandom]: wrandom,
  [Command.wotd]: wotd,
  [Command.wroll]: wroll,
  [Command.wtrade]: wtrade,
  [Command.ssearch]: ssearch,
  [Command.wdaily]: wdaily,
};

export default commands;
