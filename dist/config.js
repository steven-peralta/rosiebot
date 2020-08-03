"use strict";
var _a, _b;
Object.defineProperty(exports, "__esModule", { value: true });
const config = {
    waifuAPIKey: (_a = process.env.WAIFU_API_KEY) !== null && _a !== void 0 ? _a : '',
    discordTokenKey: (_b = process.env.DISCORD_TOKEN_KEY) !== null && _b !== void 0 ? _b : '',
    commandPrefix: '!',
    mongodbUri: process.env.MONGODB_URI,
};
exports.default = config;
//# sourceMappingURL=config.js.map