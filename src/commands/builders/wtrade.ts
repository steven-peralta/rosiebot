import { DocumentType } from '@typegoose/typegoose';
import {
  EmbedFieldData,
  Guild,
  MessageEmbedOptions,
  User as DiscordUser,
} from 'discord.js';
import UserModel, { User } from 'rosiebot/src/db/models/User';
import { Command, ErrorMessage, StatusCode } from 'rosiebot/src/util/enums';
import {
  CommandBuilder,
  CommandFormatter,
  CommandCallback,
  CommandMetadata,
  CommandProcessor,
  TradeParams,
} from 'rosiebot/src/commands/types';
import WaifuModel, { Waifu } from 'rosiebot/src/db/models/Waifu';
import APIField from 'rosiebot/src/util/APIField';
import { getUsersFromMentionsStr } from 'rosiebot/src/util/string';
import InteractiveMessage from 'rosiebot/src/discord/messages/InteractiveMessage';
import { logModuleWarning } from 'rosiebot/src/util/logger';
import {
  Button,
  ButtonCallback,
} from 'rosiebot/src/discord/messages/BaseInteractiveMessage';

interface WTradeResponse {
  target: DiscordUser;
  trade?: {
    userModel: DocumentType<User>;
    targetModel: DocumentType<User>;
    guild: Guild;
    has: { [id: number]: { name: string } };
    wants: { [id: number]: { name: string } };
  };
  errors?: {
    targetDoesntOwn?: Waifu;
    senderDoesntOwn?: Waifu;
    senderOwns?: Waifu;
    targetOwns?: Waifu;
  };
}

const metadata: CommandMetadata = {
  name: Command.wtrade,
  description: 'Used to trade waifus with another user',
  arguments: '<@user> <your waifu ids> <their waifu ids>',
  supportsDM: false,
};

const command: CommandCallback<WTradeResponse, TradeParams> = async (
  params
) => {
  const start = Date.now();
  if (params) {
    const { sender, guild, target, has, wants } = params;
    const senderModel = await UserModel.findOne({
      [APIField.userId]: sender.id,
      [APIField.serverId]: guild.id,
    });
    const targetModel = await UserModel.findOne({
      [APIField.userId]: target.id,
      [APIField.serverId]: guild.id,
    });

    if (senderModel && targetModel) {
      let senderDoesntOwn;
      let targetDoesntOwn;
      let senderOwns;
      let targetOwns;
      const satisfiesWantsConstraints = wants.every((ref) => {
        if (senderModel[APIField.ownedWaifus].includes(ref)) {
          senderOwns = WaifuModel.findById(ref).lean();
          return false;
        }
        if (!targetModel[APIField.ownedWaifus].includes(ref)) {
          targetDoesntOwn = WaifuModel.findById(ref).lean();
          return false;
        }
        return true;
      });
      const satisfiesHasConstraints = has.every((ref) => {
        if (!senderModel[APIField.ownedWaifus].includes(ref)) {
          senderDoesntOwn = WaifuModel.findById(ref).lean();
          return false;
        }
        if (targetModel[APIField.ownedWaifus].includes(ref)) {
          targetOwns = WaifuModel.findById(ref).lean();
          return false;
        }
        return true;
      });

      if (senderDoesntOwn) {
        senderDoesntOwn = await Promise.resolve(senderDoesntOwn);
      }

      if (targetDoesntOwn) {
        targetDoesntOwn = await Promise.resolve(targetDoesntOwn);
      }

      if (senderOwns) {
        senderOwns = await Promise.resolve(senderOwns);
      }

      if (targetOwns) {
        targetOwns = await Promise.resolve(targetOwns);
      }

      if (satisfiesWantsConstraints && satisfiesHasConstraints) {
        const hasObj = (
          await Promise.all(has.map((id) => WaifuModel.findById(id)))
        ).reduce((acc, val) => {
          if (val) {
            acc[val[APIField._id]] = { name: val[APIField.name] };
          }
          return acc;
        }, {} as { [id: number]: { name: string } });
        const wantsObj = (
          await Promise.all(wants.map((id) => WaifuModel.findById(id)))
        ).reduce((acc, val) => {
          if (val) {
            acc[val[APIField._id]] = { name: val[APIField.name] };
          }
          return acc;
        }, {} as { [id: number]: { name: string } });

        return {
          statusCode: StatusCode.Success,
          data: {
            target,
            trade: {
              userModel: senderModel,
              targetModel,
              guild,
              has: hasObj,
              wants: wantsObj,
            },
          },
          time: Date.now() - start,
        };
      }
      return {
        statusCode: StatusCode.Error,
        data: {
          target,
          errors: {
            senderDoesntOwn,
            targetDoesntOwn,
            senderOwns,
            targetOwns,
          },
        },
      };
    }
  }
  return {
    statusCode: StatusCode.Error,
  };
};

const processor: CommandProcessor<WTradeResponse> = async (msg, args) => {
  const { author, guild } = msg;
  if (args.length < 3) {
    return {
      statusCode: StatusCode.InsufficientArgs,
    };
  }
  if (guild) {
    const targets = getUsersFromMentionsStr(guild, [args[0]]);
    if (!targets || targets.length !== 1) {
      return {
        statusCode: StatusCode.IncorrectArgs,
      };
    }
    const [target] = targets;
    const has = args[1].split(',').map((val) => Number.parseInt(val, 10));
    const wants = args[2].split(',').map((val) => Number.parseInt(val, 10));

    return command({
      sender: author,
      guild,
      target,
      has,
      wants,
    });
  }
  return {
    statusCode: StatusCode.Error,
  };
};

const formatter: CommandFormatter<WTradeResponse, WTradeResponse> = (
  result,
  user
) => {
  const { statusCode } = result;
  if (result.data) {
    const { trade, errors, target } = result.data;
    if (statusCode === StatusCode.Error && errors) {
      if (target) {
        const {
          targetDoesntOwn,
          senderDoesntOwn,
          senderOwns,
          targetOwns,
        } = errors;
        let content = `${user}`;
        if (targetDoesntOwn) {
          content = `${content}, ${target} doesn't own ${
            targetDoesntOwn[APIField.name]
          }`;
        }
        if (senderDoesntOwn) {
          content = `${content}, you don't own ${
            senderDoesntOwn[APIField.name]
          }`;
        }
        if (senderOwns) {
          content = `${content}, you already own ${senderOwns[APIField.name]}`;
        }
        if (targetOwns) {
          content = `${content}, ${target} already owns ${
            targetOwns[APIField.name]
          }`;
        }
        return {
          content: `${content}.`,
        };
      }
      logModuleWarning(
        'We received a valid error object in WTradeResponse but no target was passed through the formatter.'
      );
      return {
        content: `${user} ${ErrorMessage[statusCode]}`,
      };
    }
    if (trade && target) {
      const singleTarget = Array.isArray(target) ? target[0] : target;
      if (singleTarget) {
        const { userModel, targetModel, has, wants } = trade;
        const hasStr = Object.values(has)
          .map((obj) => `• ${obj.name}`)
          .join('\n');
        const wantsStr = Object.values(wants)
          .map((obj) => `• ${obj.name}`)
          .join('\n');
        const hasField: EmbedFieldData = {
          name: `**Has**`,
          value: hasStr,
          inline: true,
        };
        const wantsField: EmbedFieldData = {
          name: `**Wants**`,
          value: wantsStr,
          inline: true,
        };
        const embed: MessageEmbedOptions = {
          title: 'Trade Request',
          fields: [hasField, wantsField],
        };
        const confirmationMessage = new InteractiveMessage<WTradeResponse>(
          false,
          result.data,
          embed,
          `${singleTarget}: ${user} is offering the following trade request:`,
          [singleTarget.id]
        );
        const confirmationButton: ButtonCallback<WTradeResponse> = async () => {
          const trader1Waifus = userModel[APIField.ownedWaifus];
          const trader2Waifus = targetModel[APIField.ownedWaifus];
          Object.keys(has).forEach((id) => {
            const waifuId = Number.parseInt(id, 10);
            trader1Waifus.splice(trader1Waifus.indexOf(waifuId), 1);
            trader2Waifus.push(waifuId);
          });
          Object.keys(wants).forEach((id) => {
            const waifuId = Number.parseInt(id, 10);
            trader2Waifus.splice(trader2Waifus.indexOf(waifuId), 1);
            trader1Waifus.push(waifuId);
          });
          await userModel.save();
          await targetModel.save();
          confirmationMessage.setContent(
            `${user} ${target} You accepted the trade.`
          );
          await confirmationMessage.message.suppressEmbeds(true);
          await confirmationMessage.update();
          return StatusCode.Success;
        };
        const cancelButton: ButtonCallback<WTradeResponse> = async () => {
          confirmationMessage.setContent(
            `${user} ${target} You denied the trade.`
          );
          await confirmationMessage.message.suppressEmbeds(true);
          await confirmationMessage.update();
          return StatusCode.Success;
        };
        confirmationMessage.setButtons({
          [Button.Checkmark]: confirmationButton,
          [Button.Cancel]: cancelButton,
        });
        return {
          embed: confirmationMessage,
        };
      }
    }
  }
  return {
    content: `${user} ${ErrorMessage[statusCode]}`,
  };
};

const wtrade: CommandBuilder<WTradeResponse, TradeParams, WTradeResponse> = {
  metadata,
  command,
  processor,
  formatter,
};

export default wtrade;
