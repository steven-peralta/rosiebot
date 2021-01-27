import { ColorResolvable, MessageEmbedOptions } from 'discord.js';
import discordClientInstance from 'rosiebot/src/discord/client';

const brandingColor = '#7752a0';

const brandingEmbed = (
  queryTime?: number,
  color?: ColorResolvable
): MessageEmbedOptions => {
  return {
    color: color ?? brandingColor,
    footer: {
      text: `rosiebot v${process.env.npm_package_version}${
        queryTime ? ` (${queryTime}ms)` : ''
      }`,
      iconURL: discordClientInstance.user?.displayAvatarURL(),
    },
  };
};

export default brandingEmbed;
