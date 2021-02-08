import { MessageEmbedOptions, StringResolvable, User } from 'discord.js';
import Series from '$db/models/Series';
import APIField from '$util/APIField';
import { truncateString } from '$util/string';
import PaginationMessage from '$discord/messages/PaginationMessage';
import InteractiveMessage from '$discord/messages/InteractiveMessage';

export const seriesEmbed = (series: Series): MessageEmbedOptions => {
  const {
    [APIField.mwlUrl]: url,
    [APIField.name]: name,
    [APIField.type]: type,
    [APIField.romajiName]: romajiName,
    [APIField.description]: description,
    [APIField.mwlDisplayPictureUrl]: seriesPortrait,
    [APIField.releaseDate]: releaseDate,
    [APIField.episodeCount]: episodeCount,
    [APIField.airingStart]: airingStart,
    [APIField.airingEnd]: airingEnd,
  } = series;

  const embed: MessageEmbedOptions = {
    title: `${name}${romajiName ? ` - ${romajiName}` : ''}`,
    url,
    fields: [],
  };

  if (description) {
    embed.description = truncateString(description);
  }

  if (type) {
    embed.fields?.push({
      name: 'Type',
      value: type,
      inline: true,
    });
  }

  if (seriesPortrait) {
    embed.image = {
      url: seriesPortrait,
    };
  }

  if (releaseDate) {
    embed.fields?.push({
      name: 'Release Date',
      value: releaseDate,
      inline: true,
    });
  }

  if (episodeCount) {
    embed.fields?.push({
      name: 'Episode Count',
      value: episodeCount,
      inline: true,
    });
  }

  if (airingStart && airingEnd) {
    embed.fields?.push({
      name: 'Airing Dates',
      value: `${airingStart} - ${airingEnd}`,
      inline: true,
    });
  } else if (airingStart) {
    embed.fields?.push({
      name: 'Airing Dates',
      value: `${airingStart} - `,
      inline: true,
    });
  }

  // at this point, push empty fields because flexbox will cause the last row to look
  // weird if there's not 3
  for (let i = 0; i < (embed.fields?.length ?? 0) % 3; i += 1) {
    embed.fields?.push({
      name: '\u200B',
      value: '\u200B',
      inline: true,
    });
  }

  return embed;
};

export const pagedInteractiveSeriesMessage = (
  series: Series[],
  embedOptions: MessageEmbedOptions = {},
  deleteOnTimeout = false,
  content?: StringResolvable,
  authorizedUsers?: User[]
): PaginationMessage<Series> => {
  const authorizedSnowflakes = authorizedUsers?.map((user) => user.id);
  const paginationMessage = new PaginationMessage<Series>(
    deleteOnTimeout,
    0,
    true,
    authorizedSnowflakes,
    undefined,
    ''
  );
  const interactiveEmbeds = series.map((s) => {
    return new InteractiveMessage<Series>(
      deleteOnTimeout,
      s,
      { ...seriesEmbed(s), ...embedOptions },
      content,
      authorizedSnowflakes
    );
  });
  return paginationMessage.setPages(interactiveEmbeds);
};
