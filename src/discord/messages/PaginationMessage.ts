import { Message, Snowflake, StringResolvable, User } from 'discord.js';
import { logModuleWarning } from 'rosiebot/src/util/logger';
import BaseInteractiveMessage, {
  Button,
} from 'rosiebot/src/discord/messages/BaseInteractiveMessage';
import InteractiveMessage from 'rosiebot/src/discord/messages/InteractiveMessage';
import { StatusCode } from 'rosiebot/src/util/enums';

export default class PaginationMessage<T> extends BaseInteractiveMessage<T> {
  private _currentPage: number;

  private readonly _showPaginationIndicators: boolean;

  private _pages?: InteractiveMessage<T>[];

  private _content?: StringResolvable;

  constructor(
    deleteOnTimeout = false,
    startingPage = 0,
    showPaginationIndicators = true,
    authorizedUsers?: Snowflake[],
    pages?: InteractiveMessage<T>[],
    content?: StringResolvable
  ) {
    super(deleteOnTimeout, undefined, authorizedUsers, {
      [Button.Rewind]: async () => {
        await this.goToPage(0);
        return StatusCode.Success;
      },
      [Button.Backward]: async () => {
        await this.previousPage();
        return StatusCode.Success;
      },
      [Button.Jump]: async (_instance, user) => {
        if (user) {
          const filter = (message: Message) =>
            message.author === user &&
            (/^([0-9]+)$/.test(message.content) ||
              message.content === 'cancel');
          const prompt = `${user} Type the number of the page you'd like to jump to, or type \`cancel\` to cancel.`;
          const callback = (response: Message) => {
            const { content: c } = response;
            if (c !== 'cancel') {
              const pageNum = Number.parseInt(c, 10) - 1;
              this.goToPage(pageNum);
            }
          };
          await this.awaitResponseFromUser(user, prompt, filter, callback);
          return StatusCode.Success;
        }
        return StatusCode.Error;
      },
      [Button.Forward]: async () => {
        await this.nextPage();
        return StatusCode.Success;
      },
      [Button.FastForward]: async () => {
        await this.goToPage((this._pages?.length ?? 0) - 1);
        return StatusCode.Success;
      },
    });
    this._pages = pages;
    this._currentPage = startingPage;
    this._showPaginationIndicators = showPaginationIndicators;
    this._content = content;
  }

  public setCurrentPage(currentPage: number): this {
    this._currentPage = currentPage;
    return this;
  }

  get currentPage(): number {
    return this._currentPage;
  }

  public setPages(pages: InteractiveMessage<T>[]): this {
    this._pages = pages;
    return this;
  }

  get pages(): InteractiveMessage<T>[] | undefined {
    return this._pages;
  }

  public nextPage(): Promise<void> {
    if (this._pages) {
      if (this._currentPage < this._pages.length - 1) {
        this._currentPage += 1;
      }
    }

    return this.update();
  }

  public previousPage(): Promise<void> {
    if (this._currentPage > 0) {
      this._currentPage -= 1;
    }
    return this.update();
  }

  public goToPage(page: number): Promise<void> {
    if (this._pages) {
      if (page >= 0 && page <= this._pages.length - 1) {
        this._currentPage = page;
      }
    }

    return this.update();
  }

  public setContent(content: StringResolvable): this {
    this._content = content;
    return this;
  }

  public get content(): StringResolvable {
    return this._content;
  }

  public async awaitResponseFromUser(
    user: User,
    prompt: StringResolvable,
    filter: (message: Message) => boolean,
    callback: (response: Message) => void
  ): Promise<void> {
    if (this._channel) {
      const ask = await this._channel.send(prompt);
      const awaitResponse = await this._channel.awaitMessages(filter, {
        max: 1,
        time: this._timeout,
        errors: ['time'],
      });
      const response = awaitResponse.first();
      if (!response) {
        ask.delete().catch((e) => {
          logModuleWarning(
            `Exception caught when trying to delete the pagination ask message: ${e}`
          );
        });
        return;
      }

      callback(response);

      ask.delete().catch((e) => {
        logModuleWarning(
          `Exception caught when trying to delete the pagination ask message: ${e}`
        );
      });
      response.delete().catch((e) => {
        logModuleWarning(
          `Exception caught when trying to delete the pagination response message: ${e}`
        );
      });
      await this.awaitReaction();
    }
  }

  public async update(): Promise<void> {
    const page: InteractiveMessage<T> = this._pages?.[
      this._currentPage
    ] as InteractiveMessage<T>;
    if (page) {
      if (page.data) {
        this.setData(page.data);
      }

      // assume the buttons if we have an interactive embed
      this._buttons = {
        ...this._buttons,
        ...page.buttons,
      };

      let content = `${page.content}\n${this._content}`;

      if (this._showPaginationIndicators) {
        content = `${content}\nPage ${this._currentPage + 1} out of ${
          this._pages?.length
        }`;
      }

      if (!this._message && this._channel) {
        try {
          this._message = await this._channel.send({
            embed: page.embed,
            content,
          });
        } catch (e) {
          logModuleWarning(
            `Couldn't send message during pagination update: ${e}`
          );
        }
      } else if (this._message) {
        try {
          await this._message.edit({
            embed: page.embed,
            content,
          });
        } catch (e) {
          logModuleWarning(
            `Couldn't update message during pagination update: ${e}`
          );
        }
      }
    }
  }
}
