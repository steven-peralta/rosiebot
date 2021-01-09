import {
  getModelForClass,
  prop,
  Ref,
  ReturnModelType,
} from '@typegoose/typegoose';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import ApiFields from '../../util/ApiFields';
import SeriesModel, { Series } from './Series';
import mwlApi from '../../mwl/mwlApi';

export class Waifu extends Base<number> {
  @prop()
  public [ApiFields._id]!: number;

  @prop({ unique: true, required: true })
  public [ApiFields.mwlSlug]!: string;

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
    mwlId: number | string
  ): Promise<Waifu> {
    const record =
      typeof mwlId === 'number'
        ? await this.findById(mwlId)
        : await this.findOne({ [ApiFields.mwlSlug]: mwlId });

    if (record) return record;

    const mwlWaifu = await mwlApi.getWaifu(mwlId);
    const appearances = (
      await Promise.all(
        (mwlWaifu[ApiFields.appearances] ?? [])
          .map((appearance) => appearance[ApiFields.slug])
          .map((slug) => SeriesModel.findOneOrFetchFromMwl(slug))
      )
    ).map((doc) => doc._id);

    let series;
    if (
      mwlWaifu[ApiFields.series] &&
      mwlWaifu[ApiFields.series]?.[ApiFields.slug]
    ) {
      series = (
        await SeriesModel.findOneOrFetchFromMwl(
          // we know it won't be null, this is just to get rid of typescript error here
          mwlWaifu[ApiFields.series]?.[ApiFields.slug] ?? 1
        )
      )[ApiFields._id];
    }

    return WaifuModel.create({
      [ApiFields._id]: mwlWaifu[ApiFields.id],
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
