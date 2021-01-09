import Mongoose from 'mongoose';
import config from '../src/config';
import WaifuModel from '../src/db/models/Waifu';
import ApiFields from '../src/util/ApiFields';

const { mongodbUri } = config;
const db = Mongoose.connection;

Mongoose.connect(mongodbUri, {
  useNewUrlParser: true,
  useFindAndModify: true,
  useUnifiedTopology: true,
  useCreateIndex: true,
}).catch(console.error);

async function scrape() {
  const results = [];
  const count = parseInt(process.argv[2], 10) ?? 1000;

  for (let i = 1; i < count; i += 1) {
    try {
      // eslint-disable-next-line no-await-in-loop
      const result = await WaifuModel.findOneOrFetchFromMwl(i);
      console.log(
        `${i}/${count} (${(i / count) * 100}%): ${result[ApiFields.name]}`
      );
      results.push(result);
    } catch (e) {
      console.error(`${i}/${count} (${(i / count) * 100}%): ${e.message}`);
    }
  }
  console.log('finished.');
}

db.once('open', async () => {
  console.log('Connected to database');
  scrape();
});

db.on('error', () => {
  console.log('Error connecting to database');
});
