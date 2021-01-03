import {
  getModelForClass,
  prop,
  Ref,
  DocumentType,
  ReturnModelType,
} from '@typegoose/typegoose';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import * as mongoose from 'mongoose';
import ApiFields from '../../util/ApiFields';
import { Waifu } from './Waifu';

export class User extends Base {
  @prop({ required: true, unique: true })
  [ApiFields.userId]!: string;

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
  [ApiFields.favoriteWaifu]?: Ref<Waifu>[];

  public static async findOneOrCreate(
    this: ReturnModelType<typeof User>,
    userId: string
  ) {
    const record = await this.findOne({ [ApiFields.userId]: userId });
    if (record) {
      return record;
    }
    return this.create({
      [ApiFields.userId]: userId,
      [ApiFields.coins]: 200,
      [ApiFields.created]: new Date(),
      [ApiFields.updated]: new Date(),
      [ApiFields.dailyLastClaimed]: new Date(0),
      [ApiFields.ownedWaifus]: [],
    });
  }

  public setLastUpdated(this: DocumentType<User>) {
    const now = new Date();
    if (!this[ApiFields.updated] || this[ApiFields.updated] < now) {
      this[ApiFields.updated] = now;
      this.save().catch((e) => {
        throw e;
      });
    }
  }

  public setDailyClaimed(this: DocumentType<User>) {
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

  public addWaifu(this: DocumentType<User>, id: mongoose.Types.ObjectId) {
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
