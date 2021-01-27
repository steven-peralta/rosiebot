import {
  Guild,
  Message,
  MessageEmbed,
  StringResolvable,
  User,
} from 'discord.js';
import { Command, StatusCode } from 'rosiebot/src/util/enums';
import BaseInteractiveMessage from 'rosiebot/src/discord/messages/BaseInteractiveMessage';

/**
 * command metadata includes things like a description of the command and what
 * type of arguments are needed, as well as some permissions needed to run the
 * command
 */
export interface CommandMetadata {
  name: Command;
  description: string;
  arguments?: string;
  /** Set if only the bot owner defined in the config file can run this command */
  botOwnerOnly?: boolean;
  /** Set if the user can run this command directly in the DMs of the bot */
  supportsDM?: boolean;
}

export interface CommandResult<T> {
  statusCode: StatusCode;
  data?: T;
  target?: User | User[];
  time?: number;
}

/**
 * command functions take in whatever arguments they need to run their queries
 * and return a result.
 */
export type CommandCallback<ResultType, ParamType> = (
  params?: ParamType
) => Promise<CommandResult<ResultType>>;

export interface DiscordResponseContent<T> {
  embed?: MessageEmbed | BaseInteractiveMessage<T>;
  content?: StringResolvable;
}

/**
 * command processors will take the string arguments and massage them into
 * the needed types/objects for the command function. they will also do input
 * validation. returns the results of the command
 */
export type CommandProcessor<ResultType> = (
  msg: Message,
  args: string[]
) => Promise<CommandResult<ResultType>>;

/**
 * a formatter for us to format our result into a neat little message
 */
export type CommandFormatter<ResultType, ResponseType> = (
  result: CommandResult<ResultType>,
  user: User
) => DiscordResponseContent<ResponseType>;

export interface CommandBuilder<
  ResultType,
  ParamsType extends Record<string, unknown> | undefined,
  ResponseType
> {
  metadata: CommandMetadata;
  processor: CommandProcessor<ResultType>;
  formatter: CommandFormatter<ResultType, ResponseType>;
  command: CommandCallback<ResultType, ParamsType>;
}

export interface UserParams extends Record<string, unknown> {
  sender: User;
  guild: Guild;
}

export interface TargetedUserParams extends UserParams {
  target?: User;
}

export interface TradeParams extends Required<TargetedUserParams> {
  has: number[];
  wants: number[];
}

export interface SeriesSearchParams extends Record<string, unknown> {
  text: string;
  studio?: boolean;
}

export type CommandResponseType<
  C extends CommandResult<unknown>
> = C extends CommandResult<infer T> ? T : unknown;
