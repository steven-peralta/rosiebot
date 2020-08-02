import {
  Channel,
  DMChannel,
  Message,
  MessageEmbed,
  MessageEmbedAuthor,
  NewsChannel,
  TextChannel,
} from 'discord.js';
import { Waifu } from '../waifu/api';
import { IWaifuDocument } from '../db/waifus/waifus.types';
import { Embeds } from 'discord-paginationembed';

const version = 'beta 20200802';
const footerAvatarURL =
  'https://en.gravatar.com/userimage/24330621/a37c5a0e6da08903ac0d7987cfd69362.png?size=200';

export const getBranding = (embed: MessageEmbed) => {
  embed.setFooter(
    `rosiebot ${version}. waifu data provided by MyWaifuList.moe`,
    footerAvatarURL
  );
  embed.setColor(0xf5c542);
  return embed;
};

export const getWaifuListEmbed = (
  waifus: IWaifuDocument[],
  author?: MessageEmbedAuthor
): MessageEmbed => {
  const description = waifus.reduce((acc, waifu) => {
    const series = waifu.appearsIn
      ? waifu.appearsIn[0] ?? 'No series data.'
      : 'No series data.';
    acc += `**(#${waifu.mwlId}) ${waifu.name}**: *${series}*\n`;
    return acc;
  }, '');

  const embed = new MessageEmbed({
    description,
  });

  if (author) embed.setAuthor(author.name, author.iconURL, author.url);

  return getBranding(embed);
};

export const getWaifuEmbed = (
  waifu: IWaifuDocument,
  author?: MessageEmbedAuthor
): MessageEmbed => {
  const embed = new MessageEmbed({
    image: { url: waifu.displayPictureURL },
    description: waifu.description,
    title: `${waifu.name} (#${waifu.mwlId})`,
    url: waifu.url,
  });

  if (waifu.appearsIn) {
    embed.addField('Appears In', waifu.appearsIn.join(', '));
  }

  if (waifu.originalName) {
    embed.addField('Original Name', waifu.originalName, true);
  }

  if (waifu.origin) {
    embed.addField('Origin', waifu.origin, true);
  }

  if (waifu.age) {
    embed.addField('Age', waifu.age, true);
  }

  if (waifu.bloodType) {
    embed.addField('Blood Type', waifu.bloodType, true);
  }

  if (waifu.height) {
    embed.addField('Height', waifu.height, true);
  }

  if (waifu.weight) {
    embed.addField('Weight', waifu.weight, true);
  }

  if (waifu.hip) {
    embed.addField('Hip', waifu.hip, true);
  }

  if (waifu.bust) {
    embed.addField('Bust', waifu.bust, true);
  }

  if (author) {
    embed.setAuthor(author.name, author.iconURL, author.url);
  }
  return getBranding(embed);
};

export const waifuListEmbed = (
  channel: TextChannel | DMChannel,
  waifus: IWaifuDocument[][],
  author?: MessageEmbedAuthor
) => {
  const embeds = [];
  for (const waifuDocs of waifus) {
    embeds.push(getWaifuListEmbed(waifuDocs, author));
  }
  return new Embeds().setChannel(channel).setArray(embeds);
};

export const waifuCardListEmbed = (
  channel: TextChannel | DMChannel,
  waifus: IWaifuDocument[],
  author?: MessageEmbedAuthor
) => {
  const embeds = [];
  for (const waifuDoc of waifus) {
    embeds.push(getWaifuEmbed(waifuDoc, author));
  }
  return new Embeds().setChannel(channel).setArray(embeds);
};
