import { MessageEmbedOptions, Snowflake, StringResolvable } from 'discord.js';
import { logModuleWarning } from 'rosiebot/src/util/logger';
import BaseInteractiveMessage, {
  Buttons,
} from 'rosiebot/src/discord/messages/BaseInteractiveMessage';

export default class InteractiveMessage<T> extends BaseInteractiveMessage<T> {
  private _embed?: MessageEmbedOptions;

  private _content?: StringResolvable;

  constructor(
    deleteOnTimeout = false,
    data?: T,
    embed?: MessageEmbedOptions,
    content?: StringResolvable,
    authorizedUsers?: Snowflake[],
    buttons?: Buttons<T>
  ) {
    super(deleteOnTimeout, data, authorizedUsers, buttons);
    this._embed = embed;
    this._content = content;
  }

  get embed(): MessageEmbedOptions | undefined {
    return this._embed;
  }

  public setEmbed(embed: MessageEmbedOptions): this {
    this._embed = embed;
    return this;
  }

  get content(): StringResolvable {
    return this._content;
  }

  public setContent(content: StringResolvable): this {
    this._content = content;
    return this;
  }

  async update(): Promise<void> {
    if (!this._message && this._channel) {
      try {
        this._message = await this._channel.send({
          embed: this._embed,
          content: this._content,
        });
      } catch (e) {
        logModuleWarning(
          `Couldn't send message during interactive message update: ${e}`
        );
      }
    } else if (this._message) {
      try {
        await this._message.edit({
          embed: this._embed,
          content: this._content,
        });
      } catch (e) {
        logModuleWarning(
          `Couldn't update message during interactive embed update: ${e}`
        );
      }
    }
  }
}
