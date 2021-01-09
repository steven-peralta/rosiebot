import Mongoose, { Types } from 'mongoose';
import { MongoClient } from 'mongodb';
import config from '../src/config';
import UserModel from '../src/db/models/User';
import ApiFields from '../src/util/ApiFields';
import WaifuModel from '../src/db/models/Waifu';

const { mongodbUri } = config;
const db = Mongoose.connection;
const otherDb = new MongoClient('mongodb://localhost:27017', {
  useNewUrlParser: true,
});

Mongoose.connect(mongodbUri, {
  useNewUrlParser: true,
  useFindAndModify: true,
  useUnifiedTopology: true,
  useCreateIndex: true,
}).catch(console.error);

async function start() {
  const oldUsersColl = otherDb.db('old').collection('users');
  const oldUsers = await oldUsersColl.find().toArray();

  for (const oldUser of oldUsers) {
    const newWaifuIds: Types.ObjectId[] = [];
    let coins = oldUser.coins + 2000;

    for (const oldWaifu of oldUser.ownedWaifus) {
      // eslint-disable-next-line no-await-in-loop
      const newWaifu = await WaifuModel.findOne({
        [ApiFields.mwlId]: oldWaifu,
      });
      if (newWaifu) {
        newWaifuIds.push(newWaifu._id);
      } else {
        coins += 200;
      }
    }

    // eslint-disable-next-line no-await-in-loop
    await UserModel.create({
      [ApiFields.userId]: oldUser.userId,
      [ApiFields.coins]: coins,
      [ApiFields.created]: oldUser.dateOfEntry,
      [ApiFields.updated]: oldUser.lastUpdated,
      [ApiFields.dailyLastClaimed]: oldUser.dailyLastClaimed,
      [ApiFields.ownedWaifus]: newWaifuIds,
    });
  }

  // const userData: {
  //   userId: string;
  //   waifus: Types.ObjectId[];
  //   coins: number;
  // }[] = [];
  // const oldUsersColl = otherDb.db('old').collection('users');
  // await oldUsersColl.find().forEach((oldUser) => {
  //   const newWaifuIds: Types.ObjectId[] = [];
  //   let coins = oldUser.coins + 2000;
  //
  //   // eslint-disable-next-line no-restricted-syntax
  //   for (const oldWaifuId of oldUser.ownedWaifus) {
  //     const newWaifu = await WaifuModel.findOne({
  //       [ApiFields.mwlId]: oldWaifuId,
  //     });
  //
  //     if (newWaifu) {
  //       newWaifuIds.push(newWaifu._id);
  //     } else {
  //       coins += 200;
  //     }
  //   }
  //
  //   userData.push({ userId: oldUser.userId, waifus: newWaifuIds, coins });
  // });
  //
  // console.log(userData);
  // // eslint-disable-next-line no-restricted-syntax
  // for (const oldWaifuMwlIdList of oldWaifusMwlIds) {
  //   // eslint-disable-next-line no-restricted-syntax
  //   for (const oldWaifuMwlId of oldWaifuMwlIdList) {
  //     const newWaifu = await WaifuModel.findOne({
  //       [ApiFields.mwlId]: oldWaifuMwlId,
  //     });
  //     console.log(
  //       newWaifu?.[ApiFields.name] ?? `${oldWaifuMwlId} doesn't exist`
  //     );
  //   }
  // }
}

db.once('open', () => {
  otherDb.connect((err) => {
    if (err) throw err;
    start();
  });
});
