import ApiFields from '../util/ApiFields';

export type MwlId = number;
export type MwlSlug = string;

export interface MwlStudio {
  [ApiFields.id]: MwlId;
  [ApiFields.name]: string;
  [ApiFields.originalName]?: string;
}

export interface MwlSeries {
  [ApiFields.id]: MwlId;
  [ApiFields.slug]: MwlSlug;
  [ApiFields.url]: string;
  [ApiFields.name]: string;
  [ApiFields.type]: string;
  [ApiFields.originalName]?: string;
  [ApiFields.romajiName]?: string;
  [ApiFields.description]?: string;
  [ApiFields.displayPicture]?: string;
  [ApiFields.releaseDate]?: string;
  [ApiFields.episodeCount]?: number;
  [ApiFields.airingStart]?: string;
  [ApiFields.airingEnd]?: string;
  [ApiFields.studio]?: MwlStudio;
}

export interface MwlWaifu {
  [ApiFields.id]: MwlId;
  [ApiFields.slug]: MwlSlug;
  [ApiFields.creator]: { id: number; name: string };
  [ApiFields.url]: string;
  [ApiFields.name]: string;
  [ApiFields.husbando]: boolean;
  [ApiFields.nsfw]: boolean;
  [ApiFields.likes]: number;
  [ApiFields.trash]: number;
  [ApiFields.description]?: string;
  [ApiFields.originalName]?: string;
  [ApiFields.romajiName]?: string;
  [ApiFields.displayPicture]?: string;
  [ApiFields.weight]?: number;
  [ApiFields.height]?: number;
  [ApiFields.bust]?: number;
  [ApiFields.hip]?: number;
  [ApiFields.waist]?: number;
  [ApiFields.bloodType]?: string;
  [ApiFields.origin]?: string;
  [ApiFields.age]?: number;
  [ApiFields.birthdayMonth]?: string;
  [ApiFields.birthdayDay]?: number;
  [ApiFields.birthdayYear]?: string;
  [ApiFields.popularityRank]?: number;
  [ApiFields.likeRank]?: number;
  [ApiFields.trashRank]?: number;
  [ApiFields.appearances]?: MwlSeries[];
  [ApiFields.series]?: MwlSeries;
}
