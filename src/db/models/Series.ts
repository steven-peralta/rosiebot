import {
  getModelForClass,
  prop,
  Ref,
  ReturnModelType,
} from '@typegoose/typegoose';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import ApiFields from '../../util/ApiFields';
import StudioModel, { Studio } from './Studio';
import mwlApi from '../../mwl/mwlApi';
import { MwlStudio } from '../../mwl/types';

export class Series extends Base<number> {
  @prop()
  public [ApiFields._id]!: number;

  @prop({ type: String, unique: true, required: true })
  public [ApiFields.mwlSlug]!: string;

  @prop({ required: true })
  public [ApiFields.mwlUrl]!: string;

  @prop({ required: true })
  public [ApiFields.name]!: string;

  @prop()
  public [ApiFields.type]?: string;

  @prop()
  public [ApiFields.originalName]?: string;

  @prop()
  public [ApiFields.romajiName]?: string;

  @prop()
  public [ApiFields.description]?: string;

  @prop()
  public [ApiFields.mwlDisplayPictureUrl]?: string;

  @prop()
  public [ApiFields.releaseDate]?: string;

  @prop()
  public [ApiFields.episodeCount]?: number;

  @prop()
  public [ApiFields.airingStart]?: string;

  @prop()
  public [ApiFields.airingEnd]?: string;

  @prop({ ref: () => Studio })
  public [ApiFields.studio]?: Ref<Studio, number>;

  public static async findOneOrFetchFromMwl(
    this: ReturnModelType<typeof Series>,
    mwlId: number | string
  ): Promise<Series> {
    const record =
      typeof mwlId === 'number'
        ? await this.findOne({ [ApiFields.mwlId]: mwlId })
        : await this.findOne({ [ApiFields.mwlSlug]: mwlId });

    if (record) return record;

    const mwlSeries = await mwlApi.getSeries(mwlId);
    let studio;

    if (mwlSeries[ApiFields.studio]) {
      studio = (
        await StudioModel.findOneOrCreate(
          <MwlStudio>mwlSeries[ApiFields.studio]
        )
      )[ApiFields._id];
    }

    return SeriesModel.create({
      [ApiFields._id]: mwlSeries[ApiFields.id],
      [ApiFields.mwlSlug]: mwlSeries[ApiFields.slug],
      [ApiFields.mwlUrl]: mwlSeries[ApiFields.url],
      [ApiFields.name]: mwlSeries[ApiFields.name],
      [ApiFields.type]: mwlSeries[ApiFields.type],
      [ApiFields.originalName]: mwlSeries[ApiFields.originalName],
      [ApiFields.romajiName]: mwlSeries[ApiFields.romajiName],
      [ApiFields.description]: mwlSeries[ApiFields.description],
      [ApiFields.mwlDisplayPictureUrl]: mwlSeries[ApiFields.displayPicture],
      [ApiFields.releaseDate]: mwlSeries[ApiFields.releaseDate],
      [ApiFields.episodeCount]: mwlSeries[ApiFields.episodeCount],
      [ApiFields.airingStart]: mwlSeries[ApiFields.airingStart],
      [ApiFields.airingEnd]: mwlSeries[ApiFields.airingEnd],
      [ApiFields.studio]: studio,
    });
  }
}

const SeriesModel = getModelForClass(Series);

export default SeriesModel;
