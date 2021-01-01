import { getModelForClass, prop, ReturnModelType } from '@typegoose/typegoose';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import ApiFields from '../../util/ApiFields';
import { MwlId, MwlStudio } from '../../mwl/types';

export class Studio extends Base {
  @prop({ type: Number, required: true, unique: true })
  public [ApiFields.mwlId]!: MwlId;

  @prop({ required: true })
  public [ApiFields.name]!: string;

  @prop()
  public [ApiFields.originalName]?: string;

  public static async findOneOrCreate(
    this: ReturnModelType<typeof Studio>,
    mwlStudio: MwlStudio
  ): Promise<Studio> {
    const record = await this.findOne({
      [ApiFields.mwlId]: mwlStudio[ApiFields.id],
    });

    if (record) return record;

    return StudioModel.create({
      [ApiFields.mwlId]: mwlStudio[ApiFields.id],
      [ApiFields.name]: mwlStudio[ApiFields.name],
      [ApiFields.originalName]: mwlStudio[ApiFields.originalName] ?? undefined,
    });
  }
}

const StudioModel = getModelForClass(Studio);

export default StudioModel;
