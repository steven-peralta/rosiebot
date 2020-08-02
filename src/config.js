"use strict";
var _a, _b, _c, _d, _e;
Object.defineProperty(exports, "__esModule", { value: true });
const config = {
    waifuAPIKey: (_a = process.env.WAIFU_API_KEY) !== null && _a !== void 0 ? _a : '',
    discordTokenKey: (_b = process.env.DISCORD_TOKEN_KEY) !== null && _b !== void 0 ? _b : '',
    commandPrefix: '!',
    mongo: {
        hostname: (_c = process.env.MONGO_HOSTNAME) !== null && _c !== void 0 ? _c : 'localhost',
        port: (_d = process.env.MONGO_PORT) !== null && _d !== void 0 ? _d : 27017,
        db: (_e = process.env.MONGO_DB) !== null && _e !== void 0 ? _e : 'test',
    },
};
exports.default = config;
