import { getModelForClass, prop, ReturnModelType } from '@typegoose/typegoose';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import ApiFields from '../../util/ApiFields';
import { MwlStudio } from '../../mwl/types';

export class Studio extends Base<number> {
  @prop()
  public [ApiFields._id]!: number;

  @prop({ required: true })
  public [ApiFields.name]!: string;

  @prop()
  public [ApiFields.originalName]?: string;

  public static async findOneOrCreate(
    this: ReturnModelType<typeof Studio>,
    mwlStudio: MwlStudio
  ): Promise<Studio> {
    const record = await this.findById(mwlStudio[ApiFields.id]);

    if (record) return record;

    return StudioModel.create({
      [ApiFields._id]: mwlStudio[ApiFields.id],
      [ApiFields.name]: mwlStudio[ApiFields.name],
      [ApiFields.originalName]: mwlStudio[ApiFields.originalName] ?? undefined,
    });
  }
}

const StudioModel = getModelForClass(Studio);

export default StudioModel;
