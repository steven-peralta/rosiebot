import {
  DocumentType,
  getModelForClass,
  index,
  pre,
  prop,
  ReturnModelType,
} from '@typegoose/typegoose';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import APIField from '$util/APIField';
import { MwlStudio } from '$api/mwl/types';
import { LoggingModule, logModuleError } from '$util/logger';

function setLastUpdated(this: DocumentType<Studio>) {
  const now = new Date();
  if (!this[APIField.updated] || this[APIField.updated] < now) {
    this[APIField.updated] = now;
  }
}

@pre<Studio>('save', setLastUpdated)
@index({
  [APIField.name]: 'text',
  [APIField.originalName]: 'text',
})
export default class Studio extends Base<number> {
  @prop()
  public [APIField._id]!: number;

  @prop({ required: true })
  public [APIField.name]!: string;

  @prop({ default: new Date() })
  [APIField.created]!: Date;

  @prop({ default: new Date() })
  [APIField.updated]!: Date;

  @prop()
  public [APIField.originalName]?: string;

  public static async updateFromMWL(
    this: ReturnModelType<typeof Studio>,
    mwlStudio: MwlStudio
  ): Promise<Studio | undefined> {
    try {
      const updateOpts = {
        [APIField._id]: mwlStudio[APIField.id],
        [APIField.name]: mwlStudio[APIField.name],
        [APIField.originalName]: mwlStudio[APIField.originalName],
        [APIField.updated]: new Date(),
      };

      let record = await this.findById(mwlStudio[APIField.id]);

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
    } catch (e) {
      logModuleError(
        `Exception caught when trying to find one or create Studio: ${e}`,
        LoggingModule.DB
      );
      return undefined;
    }
  }
}

export const studioModel = getModelForClass(Studio);
