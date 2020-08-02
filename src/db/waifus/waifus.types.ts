import { Model, Document } from 'mongoose';
import { Waifu } from '../../waifu/api';

export interface IWaifu {
  mwlId: number;
  name: string;
  url: string;
  originalName?: string;
  displayPictureURL?: string;
  description?: string;
  weight?: string;
  height?: string;
  bust?: string;
  hip?: string;
  waist?: string;
  bloodType?: string;
  origin?: string;
  age?: number;
  birthdayMonth?: string;
  birthdayDay?: number;
  birthdayYear?: number;
  husbando?: boolean;
  series?: string;
  appearsIn?: string[];
}

export interface IWaifuDocument extends IWaifu, Document {}

export interface IWaifuModel extends Model<IWaifuDocument> {
  findOneOrCreate: (this: IWaifuModel, waifu: Waifu) => Promise<IWaifuDocument>;
}
