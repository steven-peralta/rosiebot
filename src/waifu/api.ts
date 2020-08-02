import config from '../config';
import WaifuAPI from './WaifuAPI';

export type WaifuID = number;

export interface Studio {
  id: WaifuID;
  name: string;
  original_name: string;
}

export interface Series {
  name: string;
  original_name: string;
  romaji_name: string;
  description: string;
  slug: string;
  release_date?: string;
  airing_start?: string;
  airing_end?: string;
  episode_count?: number;
  image: string;
  url: string;
  studio: Studio;
  id: WaifuID;
}

export interface Waifu {
  id: WaifuID;
  slug: string;
  creator_id: WaifuID;
  name: string;
  original_name?: string;
  display_picture?: string;
  description?: string;
  weight?: string;
  height?: string;
  bust?: string;
  hip?: string;
  waist?: string;
  blood_type?: string;
  origin?: string;
  age?: number;
  birthday_month?: string;
  birthday_day?: number;
  birthday_year?: number;
  likes: number;
  trash: number;
  url: string;
  husbando: boolean;
  nsfw: boolean;
  popularity_rank?: number;
  like_rank?: number;
  trash_rank?: number;
  appearances?: Series[];
  series?: Series;
}

export const waifuApi = new WaifuAPI(config.waifuAPIKey);
