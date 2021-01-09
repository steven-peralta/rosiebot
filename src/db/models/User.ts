import {
  getModelForClass,
  prop,
  Ref,
  DocumentType,
  ReturnModelType,
} from '@typegoose/typegoose';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import ApiFields from '../../util/ApiFields';
import { Waifu } from './Waifu';

export class User extends Base<string> {
  @prop()
  [ApiFields._id]!: string;

  @prop({ default: 200 })
  [ApiFields.coins]!: number;

  @prop({ default: new Date() })
  [ApiFields.created]!: Date;

  @prop({ default: new Date() })
  [ApiFields.updated]!: Date;

  @prop({ default: new Date(0) })
  [ApiFields.dailyLastClaimed]!: Date;

  @prop({ ref: () => Waifu })
  [ApiFields.ownedWaifus]!: Ref<Waifu>[];

  @prop({ ref: () => Waifu })
  [ApiFields.favoriteWaifu]?: Ref<Waifu>;

  public static async findOneOrCreate(
    this: ReturnModelType<typeof User>,
    id: string
  ): Promise<User> {
    const record = await this.findById(id);
    if (record) {
      return record;
    }
    return this.create({
      [ApiFields._id]: id,
      [ApiFields.coins]: 200,
      [ApiFields.created]: new Date(),
      [ApiFields.updated]: new Date(),
      [ApiFields.dailyLastClaimed]: new Date(0),
      [ApiFields.ownedWaifus]: [],
    });
  }

  public setLastUpdated(this: DocumentType<User>): void {
    const now = new Date();
    if (!this[ApiFields.updated] || this[ApiFields.updated] < now) {
      this[ApiFields.updated] = now;
      this.save().catch((e) => {
        throw e;
      });
    }
  }

  public setDailyClaimed(this: DocumentType<User>): void {
    const now = new Date();
    if (
      !this[ApiFields.dailyLastClaimed] ||
      this[ApiFields.dailyLastClaimed] < now
    ) {
      this[ApiFields.dailyLastClaimed] = now;
      this.save().catch((e) => {
        throw e;
      });
    }
  }

  public addWaifu(this: DocumentType<User>, id: number): void {
    if (!this[ApiFields.ownedWaifus]) {
      this[ApiFields.ownedWaifus] = [id];
      this.setLastUpdated();
      // todo: do I need to save here?
    } else {
      if (this[ApiFields.ownedWaifus].includes(id)) return;
      this[ApiFields.ownedWaifus].push(id);
      this.setLastUpdated();
      // todo: do I need to save here?
    }
  }
}

const UserModel = getModelForClass(User);

export default UserModel;
