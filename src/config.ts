const config = {
  waifuAPIKey: process.env.WAIFU_API_KEY ?? '',
  discordTokenKey: process.env.DISCORD_TOKEN_KEY ?? '',
  commandPrefix: '!',
  mongo: {
    hostname: process.env.MONGO_HOSTNAME ?? 'localhost',
    port: process.env.MONGO_PORT ?? 27017,
    db: process.env.MONGO_DB ?? 'test',
  },
};
export default config;
