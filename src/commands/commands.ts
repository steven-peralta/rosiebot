import { Command } from '@util/enums';
import wcoins from '@commands/builders/wcoins';
import wowned from '@commands/builders/wowned';
import wsearch from '@commands/builders/wsearch';
import wrandom from '@commands/builders/wrandom';
import wotd from '@commands/builders/wotd';
import wroll from '@commands/builders/wroll';
import wtrade from '@commands/builders/wtrade';
import ssearch from '@commands/builders/ssearch';
import wdaily from '@commands/builders/wdaily';

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
