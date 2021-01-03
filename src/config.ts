import { Snowflake } from 'discord.js';

interface Config {
  waifuApiKey: string;
  discordTokenKey: string;
  commandPrefix: string;
  mongodbUri: string;
  randomOrgApiKey: string;
  botOwner: Snowflake;
}

const config: Config = {
  waifuApiKey: process.env.WAIFU_API_KEY ?? '',
  discordTokenKey: process.env.DISCORD_TOKEN_KEY ?? '',
  commandPrefix: '!',
  mongodbUri: process.env.MONGODB_URI ?? 'mongodb://localhost:27017/',
  randomOrgApiKey: process.env.RANDOM_ORG_API_KEY ?? '',
  botOwner: '83078959914287104',
};

export default config;
