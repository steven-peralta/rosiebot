import {
  MessageEmbedOptions,
  StringResolvable,
  User as DiscordUser,
} from 'discord.js';
import { Waifu } from 'rosiebot/src/db/models/Waifu';
import APIField from 'rosiebot/src/util/APIField';
import { truncateString } from 'rosiebot/src/util/string';
import { Series } from 'rosiebot/src/db/models/Series';
import { Button } from 'rosiebot/src/discord/messages/BaseInteractiveMessage';
import UserModel, { User } from 'rosiebot/src/db/models/User';
import { StatusCode } from 'rosiebot/src/util/enums';
import config from 'rosiebot/src/config';
import PaginationMessage from 'rosiebot/src/discord/messages/PaginationMessage';
import InteractiveMessage from 'rosiebot/src/discord/messages/InteractiveMessage';

export const waifuEmbed = (waifu: Waifu): MessageEmbedOptions => {
  const {
    [APIField._id]: id,
    [APIField.mwlDisplayPictureUrl]: waifuPortrait,
    [APIField.name]: name,
    [APIField.originalName]: originalName,
    [APIField.mwlUrl]: url,
    [APIField.description]: description,
    [APIField.likes]: likes,
    [APIField.trash]: trash,
    [APIField.weight]: weight,
    [APIField.height]: height,
    [APIField.rank]: rank,
    [APIField.tier]: tier,
    [APIField.nsfw]: nsfw,
    [APIField.bust]: bust,
    [APIField.hip]: hip,
    [APIField.waist]: waist,
    [APIField.bloodType]: bloodType,
    [APIField.origin]: origin,
    [APIField.age]: age,
    [APIField.appearances]: appearances,
    [APIField.series]: series,
  } = waifu;

  let tierString = '';
  if (tier) {
    for (let i = 0; i < tier; i += 1) {
      tierString += ':star:';
    }
  }

  const embed: MessageEmbedOptions = {
    title: `${tierString ? `${tierString}\n` : ''}${
      nsfw ? ':underage: ' : ''
    }${name}${originalName ? ` - ${originalName}` : ''} - #${id}`,
    url,
    fields: [
      {
        name: ':heart: Likes',
        value: `${likes}`,
        inline: true,
      },
      { name: ':wastebasket: Trash', value: `${trash}`, inline: true },
    ],
  };

  if (waifuPortrait) {
    embed.image = {
      url: waifuPortrait,
    };
  }

  if (description) {
    embed.description = `**Description** (may have spoilers):\n||${truncateString(
      description
    )}||`;
  }

  if (rank) {
    embed.fields?.push({
      name: ':trophy: Rank',
      value: `#${rank}`,
      inline: true,
    });
  }

  if (weight) {
    embed.fields?.push({
      name: ':scales: Weight',
      value: `${weight} kg (${Math.round(weight * 2.20462)} lbs)`,
      inline: true,
    });
  }

  if (height) {
    embed.fields?.push({
      name: ':straight_ruler: Height',
      value: `${height} cm (${Math.floor(
        (height * 0.393701) / 12
      )} ft ${Math.floor((height * 0.393701) % 12)} in)`,
      inline: true,
    });
  }

  if (bust) {
    embed.fields?.push({
      name: ':bikini: Bust',
      value: `${bust} cm`,
      inline: true,
    });
  }

  if (hip) {
    embed.fields?.push({
      name: ':pear: Hip',
      value: `${hip} cm`,
      inline: true,
    });
  }

  if (waist) {
    embed.fields?.push({
      name: ':jeans: Waist',
      value: `${waist} cm`,
      inline: true,
    });
  }

  if (origin) {
    embed.fields?.push({
      name: ':earth_americas: Origin',
      value: `||${origin}||`,
      inline: true,
    });
  }

  if (age || age === 0) {
    embed.fields?.push({
      name: ':calendar_spiral: Age',
      value: `${age}`,
      inline: true,
    });
  }

  if (bloodType) {
    embed.fields?.push({
      name: ':drop_of_blood: Blood Type',
      value: `${bloodType}`,
      inline: true,
    });
  }

  if (series && typeof series === 'object') {
    embed.fields?.push({
      name: ':book: Series',
      value: `${(<Series>series)[APIField.name]}`,
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

  // this one isn't inline so it's cool
  if (appearances) {
    const names = appearances
      .filter((appearance) => typeof appearance === 'object')
      .map((appearance) => {
        return (<Series>appearance)[APIField.name];
      })
      .join(', ');
    if (names) {
      embed.fields?.push({
        name: ':camera_with_flash: Appears In',
        value: `${names}`,
      });
    }
  }
  return embed;
};

export const interactiveWaifuMessage = (
  waifu: Waifu,
  embedOptions: MessageEmbedOptions = {},
  content?: StringResolvable,
  deleteOnTimeout = false,
  authorizedUsers?: DiscordUser[],
  showSellButton?: boolean,
  paginationMessage?: PaginationMessage<Waifu>
): InteractiveMessage<Waifu> => {
  const embed = { ...waifuEmbed(waifu), ...embedOptions };
  const interactiveMessage = new InteractiveMessage<Waifu>(
    deleteOnTimeout,
    waifu,
    embed,
    content,
    authorizedUsers?.map((user) => user.id)
  );

  if (showSellButton) {
    interactiveMessage.addButtons({
      [Button.MoneyBag]: async (instance, user) => {
        const confirmationMessage = sellConfirmation(
          waifu,
          user,
          paginationMessage ?? interactiveMessage
        );
        if (instance.channel) {
          await confirmationMessage.setChannel(instance.channel).create();
          return StatusCode.Success;
        }
        return StatusCode.Error;
      },
    });
  }

  return interactiveMessage;
};

export const pagedInteractiveWaifuMessage = (
  waifu: Waifu[],
  embedOptions: MessageEmbedOptions = {},
  deleteOnTimeout = false,
  content?: StringResolvable,
  authorizedUsers?: DiscordUser[],
  userModel?: User
): PaginationMessage<Waifu> => {
  const paginationMessage = new PaginationMessage<Waifu>(
    deleteOnTimeout,
    0,
    true,
    authorizedUsers?.map((user) => user.id),
    undefined,
    ''
  );
  const interactiveEmbeds = waifu.map((w) => {
    let showSellButton = false;
    if (userModel) {
      showSellButton = userModel[APIField.ownedWaifus].includes(
        w[APIField._id]
      );
    }
    return interactiveWaifuMessage(
      w,
      embedOptions,
      content,
      deleteOnTimeout,
      authorizedUsers,
      showSellButton,
      paginationMessage
    );
  });
  return paginationMessage.setPages(interactiveEmbeds);
};

export const sellConfirmation = (
  waifu: Waifu,
  user: DiscordUser,
  interactiveMessage?: PaginationMessage<Waifu> | InteractiveMessage<Waifu>
): InteractiveMessage<Waifu> => {
  return new InteractiveMessage<Waifu>(
    true,
    waifu,
    undefined,
    `${user} Are you sure you want to sell your ${waifu[APIField.name]} for ${
      config.sellCost
    } coins?`,
    [user.id],
    {
      [Button.Checkmark]: async (instance, _user, guild) => {
        const typedInstance = instance as InteractiveMessage<Waifu>;
        if (guild) {
          const dbUser = await UserModel.findOne({
            [APIField.userId]: user.id,
            [APIField.serverId]: guild.id,
          });
          if (dbUser) {
            const index = dbUser[APIField.ownedWaifus].indexOf(
              waifu[APIField._id]
            );
            if (index === -1) {
              return StatusCode.Error;
            }
            dbUser[APIField.ownedWaifus].splice(index, 1);
            dbUser[APIField.coins] += config.sellCost;
            await dbUser.save();
            typedInstance.cleanup();
            if (interactiveMessage instanceof PaginationMessage) {
              if (interactiveMessage.pages) {
                const i = interactiveMessage.pages.findIndex(
                  (value) => value.data === waifu
                );
                interactiveMessage.pages.splice(i, 1);
                if (interactiveMessage.pages.length === 0) {
                  interactiveMessage.cleanup();
                } else if (i === 0) {
                  await interactiveMessage.nextPage();
                } else {
                  await interactiveMessage.previousPage();
                }
              }
            } else if (interactiveMessage instanceof InteractiveMessage) {
              interactiveMessage.cleanup();
            }

            return StatusCode.Success;
          }
        }
        return StatusCode.Error;
      },
      [Button.Cancel]: async (instance) => {
        instance.cleanup();
        return StatusCode.Success;
      },
    }
  );
};
