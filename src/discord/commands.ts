import { Message, Snowflake, TextChannel } from 'discord.js';
import { UserModel } from '../db/users/users.model';
import { waifuApi } from '../waifu/api';
import {
  getWaifuEmbed,
  getWaifuListEmbed,
  waifuCardListEmbed,
  waifuListEmbed,
} from './embed';
import { getId } from './utils';
import { WaifuModel } from '../db/waifus/waifus.model';
import { IWaifuDocument } from '../db/waifus/waifus.types';

type CommandFunc = (msg: Message, ...args: string[]) => Promise<void>;

const ownedCommand: CommandFunc = async (msg: Message, ...args: string[]) => {
  const mention = msg.mentions.users.first();
  const userId = mention
    ? getId(mention.id, msg.guild?.id)
    : getId(msg.author.id, msg.guild?.id);
  const user = await UserModel.findOneOrCreate({ userId });
  const docs = await WaifuModel.find()
    .where('mwlId')
    .in(user.ownedWaifus as any[])
    .exec();

  if (args[1] && args[1] === 'list') {
    const splitDocs: IWaifuDocument[][] = [];
    while (docs.length > 0) splitDocs.push(docs.splice(0, 15));
    await waifuListEmbed(msg.channel as TextChannel, splitDocs, {
      name: mention ? mention.username : msg.author.username,
      iconURL: mention
        ? mention.avatarURL() ?? ''
        : msg.author.avatarURL() ?? '',
    })
      .build()
      .catch(() => {
        msg.channel.send(
          'An error occurred. Do I have permissions to manage messages in this channel?'
        );
      });
  } else {
    await waifuCardListEmbed(msg.channel as TextChannel, docs, {
      name: mention ? mention.username : msg.author.username,
      iconURL: mention
        ? mention.avatarURL() ?? ''
        : msg.author.avatarURL() ?? '',
    })
      .build()
      .catch(() => {
        msg.channel.send(
          'An error occurred. Do I have permissions to manage messages in this channel?'
        );
      });
  }
};

const rollCommand = async (msg: Message, ...args: string[]) => {
  const userId = getId(msg.author.id, msg.guild?.id);
  const user = await UserModel.findOneOrCreate({ userId });

  if (user.coins && user.coins >= 200) {
    const rolledWaifu = await waifuApi.getRandomWaifu();
    await user.setCoins(user.coins - 200);
    await user.addWaifu(rolledWaifu.id);

    const waifu = await WaifuModel.findOneOrCreate(rolledWaifu); // cache our waifu

    await msg.channel.send({
      content: "Here's what you claimed:",
      embed: getWaifuEmbed(waifu, {
        name: msg.author.username,
        iconURL: msg.author.avatarURL() ?? '',
      }),
    });
  } else {
    await msg.channel.send("You don't have enough coins.");
  }
};

const dailyCommand = async (msg: Message, ...args: string[]) => {
  const userId = getId(msg.author.id, msg.guild?.id);
  const user = await UserModel.findOneOrCreate({ userId });
  const now = new Date();
  if (user.dailyLastClaimed) {
    const timeSince = (now.getTime() - user.dailyLastClaimed.getTime()) / 1000;
    if (timeSince > 86400) {
      const coins = user.coins ?? 200;
      await user.setCoins(coins + 400);
      await msg.channel.send(
        `You've claimed 400 coins. You now have ${user.coins} coins.`
      );
      await user.setDailyClaimed();
    } else {
      await msg.channel.send(
        `You've already claimed your daily prize for today. Wait another ${Math.round(
          (86400 - timeSince) / 60 / 60
        )} hours.`
      );
    }
  }
};

const infoCommand = async (msg: Message, ...args: string[]) => {
  if (args[1]) {
    // console.log(await waifuApi.getWaifu(Number.parseInt(args[1])));
    msg.channel.send('fuck you');
  }
};

const commands: Record<string, CommandFunc> = {
  wowned: ownedCommand,
  wroll: rollCommand,
  wdaily: dailyCommand,
  winfo: infoCommand,
};

export default commands;
