import { Document, Model } from 'mongoose';
import { Snowflake } from 'discord.js';

export interface IUser {
  userId: string;
  coins?: number;
  ownedWaifus?: number[];
  dailyLastClaimed?: Date;
  dateOfEntry?: Date;
  lastUpdated?: Date;
}

export interface IUserDocument extends IUser, Document {
  setLastUpdated: (this: IUserDocument) => Promise<void>;
  removeWaifu: (this: IUserDocument, waifuId: number) => Promise<void>;
  addWaifu: (this: IUserDocument, waifuId: number) => Promise<void>;
  setCoins: (this: IUserDocument, coins: number) => Promise<void>;
  setDailyClaimed: (this: IUserDocument) => Promise<void>;
}

export interface IUserModel extends Model<IUserDocument> {
  findOneOrCreate: (
    this: IUserModel,
    { userId }: { userId: Snowflake }
  ) => Promise<IUserDocument>;
}
