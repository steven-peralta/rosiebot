import {
  APIMessage,
  DMChannel,
  Message,
  NewsChannel,
  TextChannel,
} from 'discord.js';
import Result from './Result';
import CommandName from './CommandName';

/**
 * command metadata includes things like a description of the command and what
 * type of arguments are needed, as well as some permissions needed to run the
 * command
 */
export interface CommandMetadata {
  name: CommandName;
  description: string;
  arguments: string;
  /** Set if only the bot owner defined in the config file can run this command */
  botOwnerOnly?: boolean;
  /** Set if the user can run this command directly in the DMs of the bot */
  supportsDM?: boolean;
}

/**
 * command processors will take the string arguments and massage them into
 * the needed types/objects for the command function. they will also do input
 * validation. returns the results of the command
 */
export type CommandProcessor = (
  msg: Message,
  args: string[]
) => Promise<Result>;

/**
 * command functions take in whatever arguments they need to run their queries
 * and return a result.
 */
export type Command = (...args: any) => Promise<Result>;

/**
 * a formatter for us to format our result into a neat little message
 */
export type CommandFormatter = (
  channel: TextChannel | NewsChannel | DMChannel,
  result: Result
) => APIMessage;

export interface CommandBuilder {
  metadata: CommandMetadata;
  processor: CommandProcessor;
  formatter: CommandFormatter;
  command: Command;
}
