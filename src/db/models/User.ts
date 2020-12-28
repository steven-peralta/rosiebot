import {
  getModelForClass,
  prop,
  Ref,
  DocumentType,
} from '@typegoose/typegoose';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import * as mongoose from 'mongoose';
import ApiFields from '../../util/ApiFields';
import { WaifuClass } from './Waifu';

class UserClass extends Base {
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

  @prop({ ref: () => WaifuClass })
  [ApiFields.ownedWaifus]: Ref<WaifuClass>[];

  public setLastUpdated(this: DocumentType<UserClass>) {
    const now = new Date();
    if (!this[ApiFields.updated] || this[ApiFields.updated] < now) {
      this[ApiFields.updated] = now;
      this.save().catch((e) => {
        throw e;
      });
    }
  }

  public setDailyClaimed(this: DocumentType<UserClass>) {
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

  public addWaifu(this: DocumentType<UserClass>, id: mongoose.Types.ObjectId) {
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

const User = getModelForClass(UserClass);

export default User;
