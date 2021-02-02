import {
  getModelForClass,
  index,
  prop,
  Ref,
  ReturnModelType,
  DocumentType,
} from '@typegoose/typegoose';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import { User as DiscordUser, Guild } from 'discord.js';
import APIField from '$util/APIField';
import Waifu from '$db/models/Waifu';
import { LoggingModule, logModuleError, logModuleInfo } from '$util/logger';

@index({ [APIField.userId]: 1, [APIField.serverId]: 1 }, { unique: true })
export default class User extends Base {
  @prop()
  [APIField.userId]!: string;

  @prop()
  [APIField.discordTag]!: string;

  @prop()
  [APIField.serverId]!: string;

  @prop()
  [APIField.serverName]!: string;

  @prop({ default: 200 })
  [APIField.coins]!: number;

  @prop({ default: new Date() })
  [APIField.created]!: Date;

  @prop({ default: new Date() })
  [APIField.updated]!: Date;

  @prop({ default: new Date(0) })
  [APIField.dailyLastClaimed]!: Date;

  @prop({ ref: () => Waifu, type: Number })
  [APIField.ownedWaifus]!: Ref<Waifu, number>[];

  @prop({ ref: () => Waifu, type: Number })
  [APIField.favoriteWaifu]?: Ref<Waifu, number>;

  public static async findOneOrCreate(
    this: ReturnModelType<typeof User>,
    user: DiscordUser,
    guild: Guild
  ): Promise<DocumentType<User> | undefined> {
    try {
      const record = await this.findOne({
        [APIField.userId]: user.id,
        [APIField.serverId]: guild.id,
      });
      if (record) {
        return record;
      }
      logModuleInfo(
        `Creating user data for ${user.tag} (${user.id}) on server ${guild.name} (${guild.id})`,
        LoggingModule.DB
      );
      return this.create({
        [APIField.userId]: user.id,
        [APIField.serverId]: guild.id,
        [APIField.discordTag]: user.tag,
        [APIField.serverName]: guild.name,
        [APIField.coins]: 200,
        [APIField.created]: new Date(),
        [APIField.updated]: new Date(),
        [APIField.dailyLastClaimed]: new Date(0),
        [APIField.ownedWaifus]: [],
      });
    } catch (e) {
      logModuleError(
        `Exception caught while finding one or creating User: ${e}`,
        LoggingModule.DB
      );
      return undefined;
    }
  }

  public setLastUpdated(this: DocumentType<User>): void {
    const now = new Date();
    if (!this[APIField.updated] || this[APIField.updated] < now) {
      this[APIField.updated] = now;
    }
  }

  public setDailyClaimed(this: DocumentType<User>): void {
    const now = new Date();
    if (
      !this[APIField.dailyLastClaimed] ||
      this[APIField.dailyLastClaimed] < now
    ) {
      this[APIField.dailyLastClaimed] = now;
    }
  }

  public addWaifu(this: DocumentType<User>, id: number): void {
    if (!this[APIField.ownedWaifus]) {
      this[APIField.ownedWaifus] = [id];
      this.setLastUpdated();
    } else {
      this[APIField.ownedWaifus].push(id);
      this.setLastUpdated();
    }
  }
}

export const userModel = getModelForClass(User);
