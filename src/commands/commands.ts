import CommandName from './CommandName';
import wcoins from './wcoins';
import { CommandBuilder } from './types';

const commands: Record<string, CommandBuilder> = {
  [CommandName.wcoins]: wcoins,
};

export default commands;
