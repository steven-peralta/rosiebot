import { StringResolvable, User as DiscordUser } from 'discord.js';
import { Waifu } from 'rosiebot/src/db/models/Waifu';
import { ErrorMessage, StatusCode } from 'rosiebot/src/util/enums';
import {
  interactiveWaifuMessage,
  pagedInteractiveWaifuMessage,
} from 'rosiebot/src/discord/embeds/waifu';
import brandingEmbed from 'rosiebot/src/discord/embeds/branding';
import { User } from 'rosiebot/src/db/models/User';
import APIField from 'rosiebot/src/util/APIField';
import { DiscordResponseContent } from 'rosiebot/src/commands/types';

export const formatWaifuResults = (
  statusCode: StatusCode,
  content: StringResolvable = '',
  data: Waifu[] | Waifu | undefined,
  sender: DiscordUser,
  userModel?: User,
  authorizedUsers?: DiscordUser[],
  time?: number
): DiscordResponseContent<Waifu> => {
  if (statusCode === StatusCode.Success) {
    if (data) {
      if (Array.isArray(data)) {
        if (data.length === 0) {
          return {
            content: `${sender} No waifus were found for your specified search query.`,
          };
        }
        if (data.length === 1) {
          const waifu = data[0];
          let showSellButton = false;
          if (userModel) {
            showSellButton = userModel[APIField.ownedWaifus].includes(
              waifu[APIField._id]
            );
          }
          return {
            embed: interactiveWaifuMessage(
              waifu,
              brandingEmbed(time),
              content,
              false,
              authorizedUsers,
              showSellButton
            ),
          };
        }
        return {
          embed: pagedInteractiveWaifuMessage(
            data,
            brandingEmbed(time),
            false,
            content,
            authorizedUsers,
            userModel
          ),
        };
      }
      let showSellButton = false;
      if (userModel) {
        showSellButton = userModel[APIField.ownedWaifus].includes(
          data[APIField._id]
        );
      }
      return {
        embed: interactiveWaifuMessage(
          data,
          brandingEmbed(time),
          content,
          false,
          authorizedUsers,
          showSellButton
        ),
      };
    }
  }
  return {
    content: `${sender} ${ErrorMessage[statusCode]}`,
  };
};
