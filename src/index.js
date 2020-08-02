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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const discord_js_1 = require("discord.js");
const config_1 = __importDefault(require("./config"));
const commands_1 = __importDefault(require("./discord/commands"));
const mongoose_1 = __importDefault(require("mongoose"));
const { discordTokenKey, commandPrefix, mongo } = config_1.default;
const discordClient = new discord_js_1.Client();
const db = mongoose_1.default.connection;
mongoose_1.default.connect(`mongodb://${mongo.hostname}:${mongo.port}/${mongo.db}`, {
    useNewUrlParser: true,
    useFindAndModify: true,
    useUnifiedTopology: true,
    useCreateIndex: true,
}).catch(console.error);
db.once('open', () => __awaiter(void 0, void 0, void 0, function* () {
    console.log('Connected to database');
}));
db.on('error', () => {
    console.log('Error connecting to database');
});
discordClient.on('message', (msg) => __awaiter(void 0, void 0, void 0, function* () {
    const { content } = msg;
    if (content[0] === commandPrefix) {
        const commandArgs = content.substr(1).split(' ');
        const command = commands_1.default[commandArgs[0]];
        if (command)
            yield command(msg, ...commandArgs).catch(console.error);
    }
}));
discordClient.login(discordTokenKey).catch(console.error);
