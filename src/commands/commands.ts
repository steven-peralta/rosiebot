import CommandName from './CommandName';
import wcoins from './wcoins';
import wuserid from './wuserid';
import { CommandBuilder } from './types';

const commands: Record<string, CommandBuilder> = {
  [CommandName.wcoins]: wcoins,
  [CommandName.wuserid]: wuserid,
};

export default commands;
