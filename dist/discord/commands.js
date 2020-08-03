"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
const users_model_1 = require("../db/users/users.model");
const api_1 = require("../waifu/api");
const embed_1 = require("./embed");
const utils_1 = require("./utils");
const waifus_model_1 = require("../db/waifus/waifus.model");
const ownedCommand = (msg, ...args) => __awaiter(void 0, void 0, void 0, function* () {
    var _a, _b, _c, _d, _e, _f;
    const mention = msg.mentions.users.first();
    const userId = mention
        ? utils_1.getId(mention.id, (_a = msg.guild) === null || _a === void 0 ? void 0 : _a.id)
        : utils_1.getId(msg.author.id, (_b = msg.guild) === null || _b === void 0 ? void 0 : _b.id);
    const user = yield users_model_1.UserModel.findOneOrCreate({ userId });
    const docs = yield waifus_model_1.WaifuModel.find()
        .where('mwlId')
        .in(user.ownedWaifus)
        .exec();
    if (args[1] && args[1] === 'list') {
        const splitDocs = [];
        while (docs.length > 0)
            splitDocs.push(docs.splice(0, 15));
        yield embed_1.waifuListEmbed(msg.channel, splitDocs, {
            name: mention ? mention.username : msg.author.username,
            iconURL: mention
                ? (_c = mention.avatarURL()) !== null && _c !== void 0 ? _c : '' : (_d = msg.author.avatarURL()) !== null && _d !== void 0 ? _d : '',
        })
            .build()
            .catch(() => {
            msg.channel.send('An error occurred. Do I have permissions to manage messages in this channel?');
        });
    }
    else {
        yield embed_1.waifuCardListEmbed(msg.channel, docs, {
            name: mention ? mention.username : msg.author.username,
            iconURL: mention
                ? (_e = mention.avatarURL()) !== null && _e !== void 0 ? _e : '' : (_f = msg.author.avatarURL()) !== null && _f !== void 0 ? _f : '',
        })
            .build()
            .catch(() => {
            msg.channel.send('An error occurred. Do I have permissions to manage messages in this channel?');
        });
    }
});
const rollCommand = (msg, ...args) => __awaiter(void 0, void 0, void 0, function* () {
    var _g, _h;
    const userId = utils_1.getId(msg.author.id, (_g = msg.guild) === null || _g === void 0 ? void 0 : _g.id);
    const user = yield users_model_1.UserModel.findOneOrCreate({ userId });
    if (user.coins && user.coins >= 200) {
        const rolledWaifu = yield api_1.waifuApi.getRandomWaifu();
        yield user.setCoins(user.coins - 200);
        yield user.addWaifu(rolledWaifu.id);
        const waifu = yield waifus_model_1.WaifuModel.findOneOrCreate(rolledWaifu); // cache our waifu
        yield msg.channel.send({
            content: "Here's what you claimed:",
            embed: embed_1.getWaifuEmbed(waifu, {
                name: msg.author.username,
                iconURL: (_h = msg.author.avatarURL()) !== null && _h !== void 0 ? _h : '',
            }),
        });
    }
    else {
        yield msg.channel.send("You don't have enough coins.");
    }
});
const dailyCommand = (msg, ...args) => __awaiter(void 0, void 0, void 0, function* () {
    var _j, _k;
    const userId = utils_1.getId(msg.author.id, (_j = msg.guild) === null || _j === void 0 ? void 0 : _j.id);
    const user = yield users_model_1.UserModel.findOneOrCreate({ userId });
    const now = new Date();
    if (user.dailyLastClaimed) {
        const timeSince = (now.getTime() - user.dailyLastClaimed.getTime()) / 1000;
        if (timeSince > 86400) {
            const coins = (_k = user.coins) !== null && _k !== void 0 ? _k : 200;
            yield user.setCoins(coins + 400);
            yield msg.channel.send(`You've claimed 400 coins. You now have ${user.coins} coins.`);
            yield user.setDailyClaimed();
        }
        else {
            yield msg.channel.send(`You've already claimed your daily prize for today. Wait another ${Math.round((86400 - timeSince) / 60 / 60)} hours.`);
        }
    }
});
const infoCommand = (msg, ...args) => __awaiter(void 0, void 0, void 0, function* () {
    if (args[1]) {
        console.log(yield api_1.waifuApi.getWaifu(Number.parseInt(args[1])));
    }
});
const commands = {
    wowned: ownedCommand,
    wroll: rollCommand,
    wdaily: dailyCommand,
    winfo: infoCommand,
};
exports.default = commands;
//# sourceMappingURL=commands.js.map