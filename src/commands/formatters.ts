import { StringResolvable, User as DiscordUser } from 'discord.js';
import { ErrorMessage, StatusCode } from '$util/enums';
import Waifu from '$db/models/Waifu';
import User from '$db/models/User';
import { DiscordResponseContent } from '$commands/types';
import APIField from '$util/APIField';
import {
  interactiveWaifuMessage,
  pagedInteractiveWaifuMessage,
} from '$discord/embeds/waifu';
import brandingEmbed from '$discord/embeds/brandingEmbed';

const formatWaifuResults = (
  statusCode: StatusCode,
  content: StringResolvable = '',
  data: Waifu[] | Waifu | undefined,
  sender: DiscordUser,
  userDoc?: User,
  authorizedUsers?: DiscordUser[],
  time?: number
): DiscordResponseContent<Waifu> => {
  if (statusCode === StatusCode.Success) {
    if (data) {
      if (Array.isArray(data)) {
        if (data.length === 1) {
          const [waifu] = data;
          let showSellButton = false;
          if (userDoc) {
            showSellButton = userDoc[APIField.ownedWaifus].includes(
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
            userDoc
          ),
        };
      }
      let showSellButton = false;
      if (userDoc) {
        showSellButton = userDoc[APIField.ownedWaifus].includes(
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

export default formatWaifuResults;
