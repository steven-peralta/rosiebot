const config = {
  waifuAPIKey: process.env.WAIFU_API_KEY ?? '',
  discordTokenKey: process.env.DISCORD_TOKEN_KEY ?? '',
  commandPrefix: '!',
  mongodbUri: process.env.MONGODB_URI as string,
};
export default config;
