import ApiFields from '../util/ApiFields';

export interface MwlStudio {
  [ApiFields.id]: number;
  [ApiFields.name]: string;
  [ApiFields.originalName]?: string;
}

export interface MwlSeries {
  [ApiFields.id]: number;
  [ApiFields.slug]: string;
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
  [ApiFields.id]: number;
  [ApiFields.slug]: string;
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
