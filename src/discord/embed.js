"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.waifuCardListEmbed = exports.waifuListEmbed = exports.getWaifuEmbed = exports.getWaifuListEmbed = exports.getBranding = void 0;
const discord_js_1 = require("discord.js");
const discord_paginationembed_1 = require("discord-paginationembed");
const version = 'beta v1.0';
const footerAvatarURL = 'https://en.gravatar.com/userimage/24330621/a37c5a0e6da08903ac0d7987cfd69362.png?size=200';
exports.getBranding = (embed) => {
    embed.setFooter(`rosiebot ${version}. waifu data provided by MyWaifuList.moe`, footerAvatarURL);
    embed.setColor(0xf5c542);
    return embed;
};
exports.getWaifuListEmbed = (waifus, author) => {
    const description = waifus.reduce((acc, waifu) => {
        var _a;
        const series = waifu.appearsIn
            ? (_a = waifu.appearsIn[0]) !== null && _a !== void 0 ? _a : 'No series data.' : 'No series data.';
        acc += `**(#${waifu.mwlId}) ${waifu.name}**: *${series}*\n`;
        return acc;
    }, '');
    const embed = new discord_js_1.MessageEmbed({
        description,
    });
    if (author)
        embed.setAuthor(author.name, author.iconURL, author.url);
    return exports.getBranding(embed);
};
exports.getWaifuEmbed = (waifu, author) => {
    const embed = new discord_js_1.MessageEmbed({
        image: { url: waifu.displayPictureURL },
        description: waifu.description,
        title: `${waifu.name} (#${waifu.mwlId})`,
        url: waifu.url,
    });
    console.log(waifu);
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
    return exports.getBranding(embed);
};
exports.waifuListEmbed = (channel, waifus, author) => {
    const embeds = [];
    for (const waifuDocs of waifus) {
        embeds.push(exports.getWaifuListEmbed(waifuDocs, author));
    }
    return new discord_paginationembed_1.Embeds().setChannel(channel).setArray(embeds);
};
exports.waifuCardListEmbed = (channel, waifus, author) => {
    const embeds = [];
    for (const waifuDoc of waifus) {
        embeds.push(exports.getWaifuEmbed(waifuDoc, author));
    }
    return new discord_paginationembed_1.Embeds().setChannel(channel).setArray(embeds);
};
