import {
  getModelForClass,
  index,
  prop,
  ReturnModelType,
} from '@typegoose/typegoose';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import APIField from 'rosiebot/src/util/APIField';
import { MwlStudio } from 'rosiebot/src/api/mwl/types';
import { LoggingModule, logModuleError } from 'rosiebot/src/util/logger';

@index({
  [APIField.name]: 'text',
  [APIField.originalName]: 'text',
})
export class Studio extends Base<number> {
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

      return StudioModel.create({
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

const StudioModel = getModelForClass(Studio);

export default StudioModel;
