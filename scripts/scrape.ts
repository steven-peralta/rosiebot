import Mongoose from 'mongoose';
import config from '../src/config';
import Waifu from '../src/db/models/Waifu';
import ApiFields from '../src/util/ApiFields';

const { mongodbUri } = config;
const db = Mongoose.connection;

Mongoose.connect(mongodbUri, {
  useNewUrlParser: true,
  useFindAndModify: true,
  useUnifiedTopology: true,
  useCreateIndex: true,
}).catch(console.error);

const timer = (ms: number) => new Promise((res) => setTimeout(res, ms));

async function scrape() {
  const results = [];
  for (let i = 1; i < 1000; i += 1) {
    try {
      // eslint-disable-next-line no-await-in-loop
      const result = await Waifu.findOneOrFetchFromMwl(i);
      console.log(`${i}: ${result[ApiFields.name]}`);
      results.push(result);
    } catch (e) {
      console.error(`${i}: ${e.message}`);
      if (e.response && e.response.status === 429) {
        console.error('Rate limited, waiting 5 seconds');
        // eslint-disable-next-line no-await-in-loop
        await timer(1000 * 5);
        i -= 1;
      }
      continue;
    }
  }
}

db.once('open', async () => {
  console.log('Connected to database');
  await scrape();
});

db.on('error', () => {
  console.log('Error connecting to database');
});
