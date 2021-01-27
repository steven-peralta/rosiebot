import APIField from 'rosiebot/src/util/APIField';

export interface MwlStudio {
  [APIField.id]: number;
  [APIField.name]: string;
  [APIField.originalName]?: string;
}

export interface MwlSeries {
  [APIField.id]: number;
  [APIField.slug]: string;
  [APIField.url]: string;
  [APIField.name]: string;
  [APIField.type]: string;
  [APIField.originalName]?: string;
  [APIField.romajiName]?: string;
  [APIField.description]?: string;
  [APIField.displayPicture]?: string;
  [APIField.releaseDate]?: string;
  [APIField.episodeCount]?: number;
  [APIField.airingStart]?: string;
  [APIField.airingEnd]?: string;
  [APIField.studio]?: MwlStudio;
}

export interface MwlWaifu {
  [APIField.id]: number;
  [APIField.slug]: string;
  [APIField.creator]: { id: number; name: string };
  [APIField.url]: string;
  [APIField.name]: string;
  [APIField.husbando]: boolean;
  [APIField.nsfw]: boolean;
  [APIField.likes]: number;
  [APIField.trash]: number;
  [APIField.description]?: string;
  [APIField.originalName]?: string;
  [APIField.romajiName]?: string;
  [APIField.displayPicture]?: string;
  [APIField.weight]?: number;
  [APIField.height]?: number;
  [APIField.bust]?: number;
  [APIField.hip]?: number;
  [APIField.waist]?: number;
  [APIField.bloodType]?: string;
  [APIField.origin]?: string;
  [APIField.age]?: number;
  [APIField.birthdayMonth]?: string;
  [APIField.birthdayDay]?: number;
  [APIField.birthdayYear]?: string;
  [APIField.popularityRank]?: number;
  [APIField.likeRank]?: number;
  [APIField.trashRank]?: number;
  [APIField.appearances]?: MwlSeries[];
  [APIField.series]?: MwlSeries;
}
