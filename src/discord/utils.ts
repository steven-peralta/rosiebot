import { Guild, Snowflake, User } from 'discord.js';

export const getId = (userId: Snowflake, serverId?: Snowflake): string =>
  `${userId}${serverId}`;

/**
 * Given a guild object and a list of mention strings, gets the members
 * of the guild and returns a list of their user objects in the order
 * of their mention
 * @param guild
 * @param mentionStrs
 */
export const getUsersFromMentionsStr = (
  guild: Guild,
  mentionStrs: string[]
): User[] | undefined => {
  return mentionStrs.reduce((acc: User[], mentionStr) => {
    const matches = mentionStr.match(/^<@!?(\d+)>$/);
    if (matches) {
      const member = guild.member(matches[1]);
      if (member) {
        acc.push(member.user);
      }
    }
    return acc;
  }, []);
};
