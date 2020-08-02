import WaifuSchema from './waifus.schema';
import { IWaifuDocument, IWaifuModel } from './waifus.types';
import { model } from 'mongoose';

export const WaifuModel = model<IWaifuDocument>(
  'waifu',
  WaifuSchema
) as IWaifuModel;
