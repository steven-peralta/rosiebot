import {
  DocumentType,
  getModelForClass,
  index,
  pre,
  prop,
  Ref,
  ReturnModelType,
} from '@typegoose/typegoose';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import APIField from '$util/APIField';
import Studio, { studioModel } from '$db/models/Studio';
import { MwlStudio } from '$api/mwl/types';
import mwl from '$api/mwl/mwl';
import { LoggingModule, logModuleError, logModuleInfo } from '$util/logger';

function setLastUpdated(this: DocumentType<Series>) {
  const now = new Date();
  if (!this[APIField.updated] || this[APIField.updated] < now) {
    this[APIField.updated] = now;
  }
}

@pre<Series>('save', setLastUpdated)
@index({
  [APIField.name]: 'text',
  [APIField.originalName]: 'text',
  [APIField.romajiName]: 'text',
})
export default class Series extends Base<number> {
  @prop()
  public [APIField._id]!: number;

  @prop({ type: String, unique: true, required: true })
  public [APIField.mwlSlug]!: string;

  @prop({ required: true })
  public [APIField.mwlUrl]!: string;

  @prop({ required: true })
  public [APIField.name]!: string;

  @prop({ default: new Date() })
  [APIField.created]!: Date;

  @prop({ default: new Date() })
  [APIField.updated]!: Date;

  @prop()
  public [APIField.type]?: string;

  @prop()
  public [APIField.originalName]?: string;

  @prop()
  public [APIField.romajiName]?: string;

  @prop()
  public [APIField.description]?: string;

  @prop()
  public [APIField.mwlDisplayPictureUrl]?: string;

  @prop()
  public [APIField.releaseDate]?: string;

  @prop()
  public [APIField.episodeCount]?: number;

  @prop()
  public [APIField.airingStart]?: string;

  @prop()
  public [APIField.airingEnd]?: string;

  @prop({ ref: () => Studio, type: Number })
  public [APIField.studio]?: Ref<Studio, number>;

  public static async updateFromMWL(
    this: ReturnModelType<typeof Series>,
    mwlId: number | string
  ): Promise<Series | undefined> {
    try {
      const mwlSeries = await mwl.getSeries(mwlId);

      if (mwlSeries) {
        let studio;

        if (mwlSeries[APIField.studio]) {
          studio = await studioModel.updateFromMWL(
            <MwlStudio>mwlSeries[APIField.studio]
          );

          if (studio) {
            studio = studio[APIField._id];
          }
        }

        logModuleInfo(
          `Caching series data for ${mwlSeries[APIField.name]}`,
          LoggingModule.DB
        );

        const updateOpts = {
          [APIField._id]: mwlSeries[APIField.id],
          [APIField.mwlSlug]: mwlSeries[APIField.slug],
          [APIField.mwlUrl]: mwlSeries[APIField.url],
          [APIField.name]: mwlSeries[APIField.name],
          [APIField.type]: mwlSeries[APIField.type],
          [APIField.originalName]: mwlSeries[APIField.originalName],
          [APIField.romajiName]: mwlSeries[APIField.romajiName],
          [APIField.description]: mwlSeries[APIField.description],
          [APIField.mwlDisplayPictureUrl]: mwlSeries[APIField.displayPicture],
          [APIField.releaseDate]: mwlSeries[APIField.releaseDate],
          [APIField.episodeCount]: mwlSeries[APIField.episodeCount],
          [APIField.airingStart]: mwlSeries[APIField.airingStart],
          [APIField.airingEnd]: mwlSeries[APIField.airingEnd],
          [APIField.studio]: studio,
          [APIField.updated]: new Date(),
        };

        let record = await this.findById(mwlSeries[APIField.id]);
        if (record) {
          await record.updateOne(updateOpts);
        } else {
          // FIXME need to update the typegoose types to allow this
          // eslint-disable-next-line @typescript-eslint/ban-ts-comment
          // @ts-ignore
          record = await this.create({
            ...updateOpts,
            [APIField.created]: new Date(),
          });
        }

        return record;
      }
      return undefined;
    } catch (e) {
      logModuleError(
        `Exception caught when trying to find one or create Series: ${e}`,
        LoggingModule.DB
      );
      return undefined;
    }
  }
}

export const seriesModel = getModelForClass(Series);
