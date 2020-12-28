import { Snowflake } from 'discord.js';

const getId = (userId: Snowflake, serverId?: Snowflake): number =>
  Array.from(userId + (serverId ?? '')).reduce(
    (s, c) => (Math.imul(31, s) + c.charCodeAt(0)) | 0,
    0
  );

export default getId;
