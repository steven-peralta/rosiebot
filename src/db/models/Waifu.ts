import {
  getModelForClass,
  prop,
  Ref,
  ReturnModelType,
} from '@typegoose/typegoose';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import { MwlId, MwlSlug } from '../../mwl/types';
import ApiFields from '../../util/ApiFields';
import SeriesModel, { Series } from './Series';
import api from '../../mwl/api';

export class Waifu extends Base {
  @prop({ unique: true, required: true })
  public [ApiFields.mwlId]!: MwlId;

  @prop({ unique: true, required: true })
  public [ApiFields.mwlSlug]!: MwlSlug;

  @prop({ required: true })
  public [ApiFields.mwlCreatorId]!: number;

  @prop({ required: true })
  public [ApiFields.mwlCreatorName]!: string;

  @prop({ required: true })
  [ApiFields.mwlUrl]!: string;

  @prop({ required: true })
  [ApiFields.name]!: string;

  @prop({ required: true })
  [ApiFields.husbando]!: boolean;

  @prop({ required: true })
  [ApiFields.nsfw]!: boolean;

  @prop({ required: true })
  [ApiFields.likes]!: number;

  @prop({ required: true })
  [ApiFields.trash]!: number;

  @prop()
  [ApiFields.description]?: string;

  @prop()
  [ApiFields.originalName]?: string;

  @prop()
  [ApiFields.romajiName]?: string;

  @prop()
  [ApiFields.mwlDisplayPictureUrl]?: string;

  @prop()
  [ApiFields.weight]?: number;

  @prop()
  [ApiFields.height]?: number;

  @prop()
  [ApiFields.bust]?: number;

  @prop()
  [ApiFields.hip]?: number;

  @prop()
  [ApiFields.waist]?: number;

  @prop()
  [ApiFields.bloodType]?: string;

  @prop()
  [ApiFields.origin]?: string;

  @prop()
  [ApiFields.age]?: number;

  @prop()
  [ApiFields.birthdayMonth]?: string;

  @prop()
  [ApiFields.birthdayDay]?: number;

  @prop()
  [ApiFields.birthdayYear]?: string;

  @prop()
  [ApiFields.popularityRank]?: number;

  @prop()
  [ApiFields.likeRank]?: number;

  @prop()
  [ApiFields.trashRank]?: number;

  @prop({ ref: () => Series })
  [ApiFields.appearances]?: Ref<Series>[];

  @prop({ ref: () => Series })
  [ApiFields.series]?: Ref<Series>;

  public static async findOneOrFetchFromMwl(
    this: ReturnModelType<typeof Waifu>,
    mwlId: MwlId | MwlSlug
  ): Promise<Waifu> {
    const record =
      typeof mwlId === 'number'
        ? await this.findOne({ [ApiFields.mwlId]: mwlId })
        : await this.findOne({ [ApiFields.mwlSlug]: mwlId });

    if (record) return record;

    const mwlWaifu = await api.getWaifu(mwlId);
    const appearances = (
      await Promise.all(
        (mwlWaifu[ApiFields.appearances] ?? [])
          .map((appearance) => appearance[ApiFields.slug])
          .map((slug) => SeriesModel.findOneOrFetchFromMwl(slug))
      )
    ).map((doc) => doc._id);
    const series = mwlWaifu[ApiFields.series]
      ? (
          await SeriesModel.findOneOrFetchFromMwl(
            mwlWaifu[ApiFields.series]![ApiFields.slug]
          )
        )._id
      : undefined;

    return WaifuModel.create({
      [ApiFields.mwlId]: mwlWaifu[ApiFields.id],
      [ApiFields.mwlSlug]: mwlWaifu[ApiFields.slug],
      [ApiFields.mwlCreatorId]: mwlWaifu[ApiFields.creator].id,
      [ApiFields.mwlCreatorName]: mwlWaifu[ApiFields.creator].name,
      [ApiFields.mwlUrl]: mwlWaifu[ApiFields.url],
      [ApiFields.name]: mwlWaifu[ApiFields.name],
      [ApiFields.husbando]: mwlWaifu[ApiFields.husbando],
      [ApiFields.nsfw]: mwlWaifu[ApiFields.nsfw],
      [ApiFields.likes]: mwlWaifu[ApiFields.likes],
      [ApiFields.trash]: mwlWaifu[ApiFields.trash],
      [ApiFields.description]: mwlWaifu[ApiFields.description],
      [ApiFields.originalName]: mwlWaifu[ApiFields.originalName],
      [ApiFields.romajiName]: mwlWaifu[ApiFields.romajiName],
      [ApiFields.mwlDisplayPictureUrl]: mwlWaifu[ApiFields.displayPicture],
      [ApiFields.weight]: mwlWaifu[ApiFields.weight],
      [ApiFields.height]: mwlWaifu[ApiFields.height],
      [ApiFields.bust]: mwlWaifu[ApiFields.bust],
      [ApiFields.hip]: mwlWaifu[ApiFields.hip],
      [ApiFields.waist]: mwlWaifu[ApiFields.waist],
      [ApiFields.bloodType]: mwlWaifu[ApiFields.bloodType],
      [ApiFields.origin]: mwlWaifu[ApiFields.origin],
      [ApiFields.age]: mwlWaifu[ApiFields.age],
      [ApiFields.birthdayMonth]: mwlWaifu[ApiFields.birthdayMonth],
      [ApiFields.birthdayDay]: mwlWaifu[ApiFields.birthdayDay],
      [ApiFields.birthdayYear]: mwlWaifu[ApiFields.birthdayYear],
      [ApiFields.popularityRank]: mwlWaifu[ApiFields.popularityRank],
      [ApiFields.likeRank]: mwlWaifu[ApiFields.likeRank],
      [ApiFields.trashRank]: mwlWaifu[ApiFields.trashRank],
      [ApiFields.appearances]: appearances,
      [ApiFields.series]: series,
    });
  }
}

const WaifuModel = getModelForClass(Waifu);

export default WaifuModel;
