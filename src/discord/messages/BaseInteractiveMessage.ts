import {
  DMChannel,
  Guild,
  Message,
  MessageReaction,
  Snowflake,
  TextChannel,
  User,
} from 'discord.js';
import {
  logModuleError,
  logModuleInfo,
  logModuleWarning,
} from 'rosiebot/src/util/logger';
import { EventEmitter } from 'events';
import { StatusCode } from 'rosiebot/src/util/enums';

export enum Button {
  Forward = '▶️',
  Backward = '◀️',
  FastForward = '⏭',
  Rewind = '⏮',
  Jump = '⤴️',
  Checkmark = '✅',
  Cancel = '🚫',
  MoneyBag = '💰',
}

export type Buttons<T> = Partial<Record<Button, ButtonCallback<T>>>;
export type ButtonCallback<T> = (
  instance: BaseInteractiveMessage<T>,
  sender: User,
  guild?: Guild,
  data?: T
) => Promise<StatusCode>;
export default abstract class BaseInteractiveMessage<T> extends EventEmitter {
  protected _deleteOnTimeout: boolean;

  protected _timeout: number;

  protected _deleted: boolean;

  protected _buttons?: Buttons<T>;

  protected _data?: T;

  protected _channel?: TextChannel | DMChannel;

  protected _message?: Message;

  protected _authorizedUsers?: Snowflake[];

  constructor(
    deleteOnTimeout = false,
    data?: T,
    authorizedUsers?: Snowflake[],
    buttons?: Buttons<T>
  ) {
    super();
    this._deleteOnTimeout = deleteOnTimeout;
    this._data = data;
    this._buttons = buttons;
    this._authorizedUsers = authorizedUsers;
    this._timeout = 30000;
    this._deleted = false;
  }

  public setAuthorizedUsers(authorizedUsers: Snowflake[]): this {
    this._authorizedUsers = authorizedUsers;
    return this;
  }

  get authorizedUsers(): Snowflake[] | undefined {
    return this._authorizedUsers;
  }

  public setTimeout(timeout: number): this {
    this._timeout = timeout;
    return this;
  }

  get timeout(): number {
    return this._timeout;
  }

  public setButtons(buttons: Buttons<T>): this {
    this._buttons = buttons;
    return this;
  }

  public addButtons(buttons: Buttons<T>): this {
    this._buttons = { ...this._buttons, ...buttons };
    return this;
  }

  get buttons(): Buttons<T> | undefined {
    return this._buttons;
  }

  get isDeleted(): boolean {
    return this._deleted;
  }

  public setChannel(channel: TextChannel | DMChannel): this {
    this._channel = channel;
    return this;
  }

  get channel(): TextChannel | DMChannel | undefined {
    return this._channel;
  }

  public setMessage(message: Message): this {
    this._message = message;
    if (
      message.channel instanceof TextChannel ||
      message.channel instanceof DMChannel
    ) {
      this._channel = message.channel;
    }
    return this;
  }

  get message(): Message {
    return <Message>this._message;
  }

  public setData(data: T): this {
    this._data = data;
    return this;
  }

  get data(): T | undefined {
    return this._data;
  }

  public async awaitReaction(): Promise<void> {
    if (!this._buttons || Object.keys(this._buttons).length === 0) {
      logModuleWarning(
        'Interactive embed tried awaiting for reactions when it has no buttons set.'
      );
      return;
    }
    if (this._message) {
      try {
        const awaitReactions = await this._message.awaitReactions(
          (reaction: MessageReaction) =>
            Object.keys(this._buttons ?? {}).includes(reaction.emoji.name),
          {
            max: 1,
            time: this._timeout,
            errors: ['time'],
          }
        );
        const reaction = awaitReactions.first();
        if (!reaction) {
          logModuleInfo(
            'No one responded to our interactive message. Cleaning up...'
          );
          this.cleanup();
          return;
        }
        const user = reaction.users.cache.last();
        const { name: emojiName } = reaction.emoji;

        reaction.users.remove(user).catch((e) => {
          logModuleWarning(
            `Couldn't remove user from reaction. Do I have MANAGE_MESSAGES permission? ${e}`
          );
        });

        const buttonFunction = this._buttons[emojiName as Button];

        if (buttonFunction && user) {
          if (
            this._authorizedUsers &&
            !this._authorizedUsers.includes(user.id)
          ) {
            logModuleInfo(
              `${user.tag} tried to interactive with interactive message that they are not authorized to use.`
            );
            await this.awaitReaction();
            return;
          }
          await buttonFunction(
            this,
            user,
            this._message.guild ?? undefined,
            this._data
          );
        }

        if (!this._deleted) {
          await this.awaitReaction();
        }

        return;
      } catch (e) {
        this.cleanup();
      }
    }
  }

  public cleanup(): void {
    if (this._message) {
      this._message.reactions.removeAll().then((message) => {
        if (this._deleteOnTimeout && message.deletable) {
          message.delete().catch((e) => {
            logModuleWarning(`Couldn't delete interactive message: ${e}`);
          });
        }
      });
    }
    this._deleted = true;
  }

  public checkPermissions(): boolean {
    if (this._channel && this._channel.client.user) {
      if (this._channel instanceof TextChannel) {
        const perms = this._channel
          .permissionsFor(this._channel.client.user)
          ?.missing([
            'ADD_REACTIONS',
            'EMBED_LINKS',
            'VIEW_CHANNEL',
            'SEND_MESSAGES',
          ]);
        return !(perms && perms.length);
      }
      return true;
    }
    return false;
  }

  public async drawEmojis(): Promise<void> {
    if (!this._buttons || Object.keys(this._buttons).length === 0) {
      logModuleWarning(
        'Interactive embed tried drawing emojis when it has no buttons set.'
      );
      return;
    }
    if (this._message) {
      const { reactions } = this._message;
      // filter out emojis already drawn on our message
      const promises: (Promise<MessageReaction> | undefined)[] = Object.keys(
        this._buttons
      )
        .filter((emoji) => !reactions.cache.keyArray().includes(emoji))
        .map((emojiButton) => this._message?.react(emojiButton));

      reactions.cache
        .keyArray()
        .filter((emoji) => !Object.keys(this._buttons ?? {}).includes(emoji))
        .forEach((emoji) => {
          const reaction = reactions.cache.get(emoji);
          if (reaction) {
            promises.push(reaction.remove());
          }
        });

      await Promise.all(promises).catch((e) => {
        logModuleError(
          `Error while drawing emojis for interactive embed: ${e}`
        );
      });
    }
  }

  public async create(): Promise<void> {
    if (!this._channel) {
      logModuleError('Channel has not been set for the interactive embed');
    }
    if (this.checkPermissions()) {
      await this.update();
      await this.drawEmojis();
      this.awaitReaction().catch((e) => {
        logModuleWarning(
          `Exception caught while invoking await reactions in our interactive embed: ${e}`
        );
      });
    } else {
      logModuleWarning(
        `We are missing permissions to build the interactive embed`
      );
    }
  }

  abstract async update(): Promise<void>;
}
