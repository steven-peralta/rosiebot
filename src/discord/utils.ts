import { Snowflake } from 'discord.js';

export const getId = (userId: Snowflake, serverId?: Snowflake) =>
  userId + (serverId ?? '');
