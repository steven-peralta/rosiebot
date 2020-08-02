import { Schema } from 'mongoose';
import { findOneOrCreate } from './waifus.statics';

const WaifuSchema = new Schema({
  mwlId: {
    type: Number,
    required: true,
  },
  name: {
    type: String,
    required: true,
  },
  url: {
    type: String,
    required: true,
  },
  originalName: String,
  displayPictureURL: String,
  description: String,
  weight: String,
  height: String,
  bust: String,
  hip: String,
  waist: String,
  bloodType: String,
  origin: String,
  age: Number,
  birthdayMonth: String,
  birthdayDay: Number,
  birthdayYear: Number,
  husbando: Boolean,
  series: String,
  appearsIn: [String],
});

WaifuSchema.statics.findOneOrCreate = findOneOrCreate;

export default WaifuSchema;
