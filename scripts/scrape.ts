/* eslint-disable no-console, no-await-in-loop */
import config from '@config';
import Mongoose from 'mongoose';

import { waifuModel } from '@db/models/Waifu';
import APIField from '@util/APIField';

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
  const count = parseInt(process.argv[2], 10) ?? 1000;

  for (let i = 1; i < count; i += 1) {
    try {
      const result = await waifuModel.findOneOrFetchFromMwl(i);
      if (result) {
        console.log(
          `${i}/${count} (${Math.round((i / count) * 100)}%): ${
            result[APIField.name]
          }`
        );
        results.push(result);
      }
    } catch (e) {
      console.error(
        `${i}/${count} (${Math.round((i / count) * 100)}%): ${e.message}`
      );
      if (e.response && e.response.status === 429) {
        console.error('Rate limited, waiting 5 seconds');
        await timer(1000 * 5);
        i -= 1;
      }
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
