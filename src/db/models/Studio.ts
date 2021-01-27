import APIField from '@util/APIField';
import {
  getModelForClass,
  index,
  prop,
  ReturnModelType,
} from '@typegoose/typegoose';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import { MwlStudio } from '@api/mwl/types';
import { LoggingModule, logModuleError } from '@util/logger';

@index({
  [APIField.name]: 'text',
  [APIField.originalName]: 'text',
})
export default class Studio extends Base<number> {
  @prop()
  public [APIField._id]!: number;

  @prop({ required: true })
  public [APIField.name]!: string;

  @prop()
  public [APIField.originalName]?: string;

  public static async findOneOrCreate(
    this: ReturnModelType<typeof Studio>,
    mwlStudio: MwlStudio
  ): Promise<Studio | undefined> {
    try {
      const record = await this.findById(mwlStudio[APIField.id]);

      if (record) return record;

      return studioModel.create({
        [APIField._id]: mwlStudio[APIField.id],
        [APIField.name]: mwlStudio[APIField.name],
        [APIField.originalName]: mwlStudio[APIField.originalName],
      });
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
