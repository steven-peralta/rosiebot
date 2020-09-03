import { Schema } from 'mongoose';
import { findOneOrCreate } from './users.statics';
import {
  addWaifu,
  removeWaifu,
  setCoins,
  setDailyClaimed,
  setLastUpdated,
} from './users.methods';

const UserSchema = new Schema({
  userId: {
    type: String,
    required: true,
    unique: true,
  },
  coins: {
    type: Number,
    default: 200,
  },
  ownedWaifus: [{ type: Number }],
  dailyLastClaimed: {
    type: Date,
    default: 0,
  },
  dateOfEntry: {
    type: Date,
    default: new Date(),
  },
  lastUpdated: {
    type: Date,
    default: new Date(),
  },
});

UserSchema.statics.findOneOrCreate = findOneOrCreate;

UserSchema.methods.setLastUpdated = setLastUpdated;
UserSchema.methods.removeWaifu = removeWaifu;
UserSchema.methods.addWaifu = addWaifu;
UserSchema.methods.setCoins = setCoins;
UserSchema.methods.setDailyClaimed = setDailyClaimed;

export default UserSchema;
