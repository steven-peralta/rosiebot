import {
  DocumentType,
  getModelForClass,
  index,
  prop,
  Ref,
  ReturnModelType,
} from '@typegoose/typegoose';
import APIField from '@util/APIField';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import Series, { seriesModel } from '@db/models/Series';
import mwl from '@api/mwl/mwl';
import { LoggingModule, logModuleError, logModuleInfo } from '@util/logger';
import { FilterQuery, QueryFindOptions } from 'mongoose';
import randomOrg from '@api/random-org/randomOrg';
import { Tier } from '@util/enums';
import { QueryOptions } from '@db/types';
import hash from 'object-hash';
import Timeout = NodeJS.Timeout;

export const cachedQueries: Record<string, Waifu[]> = {};
export const timeouts: Record<string, Timeout> = {};
@index({
  [APIField.name]: 'text',
  [APIField.originalName]: 'text',
  [APIField.romajiName]: 'text',
})
export default class Waifu extends Base<number> {
  @prop()
  public [APIField._id]!: number;

  @prop({ unique: true, required: true })
  public [APIField.mwlSlug]!: string;

  @prop({ required: true })
  public [APIField.mwlCreatorId]!: number;

  @prop({ required: true })
  public [APIField.mwlCreatorName]!: string;

  @prop({ required: true })
  [APIField.mwlUrl]!: string;

  @prop({ required: true })
  [APIField.name]!: string;

  @prop({ required: true })
  [APIField.husbando]!: boolean;

  @prop({ required: true })
  [APIField.nsfw]!: boolean;

  @prop({ required: true })
  [APIField.likes]!: number;

  @prop({ required: true })
  [APIField.trash]!: number;

  @prop()
  [APIField.description]?: string;

  @prop()
  [APIField.originalName]?: string;

  @prop()
  [APIField.romajiName]?: string;

  @prop()
  [APIField.mwlDisplayPictureUrl]?: string;

  @prop()
  [APIField.weight]?: number;

  @prop()
  [APIField.height]?: number;

  @prop()
  [APIField.bust]?: number;

  @prop()
  [APIField.hip]?: number;

  @prop()
  [APIField.waist]?: number;

  @prop()
  [APIField.bloodType]?: string;

  @prop()
  [APIField.origin]?: string;

  @prop()
  [APIField.age]?: number;

  @prop()
  [APIField.birthdayMonth]?: string;

  @prop()
  [APIField.birthdayDay]?: number;

  @prop()
  [APIField.birthdayYear]?: string;

  @prop()
  [APIField.score]?: number;

  @prop()
  [APIField.rank]?: number;

  @prop()
  [APIField.tier]?: number;

  @prop({ ref: () => Series, type: Number })
  [APIField.appearances]?: Ref<Series, number>[];

  @prop({ ref: () => Series, type: Number })
  [APIField.series]?: Ref<Series, number>;

  public static async findOneOrFetchFromMwl(
    this: ReturnModelType<typeof Waifu>,
    mwlId: number | string,
    updateScores = false
  ): Promise<Waifu | undefined> {
    try {
      const record =
        typeof mwlId === 'number'
          ? await this.findById(mwlId)
          : await this.findOne({ [APIField.mwlSlug]: mwlId });

      if (record) return record;

      const mwlWaifu = await mwl.getWaifu(mwlId);

      if (mwlWaifu) {
        const appearances = (
          await Promise.all(
            (mwlWaifu[APIField.appearances] ?? [])
              .map((appearance) => appearance[APIField.slug])
              .map((slug) => seriesModel.findOneOrFetchFromMwl(slug))
          )
        ).map((doc) => doc?.[APIField._id] ?? 0); // if the series doc some how ends up being undefined, use a predetermined id value

        let series;
        if (
          mwlWaifu[APIField.series] &&
          mwlWaifu[APIField.series]?.[APIField.slug]
        ) {
          series = await seriesModel.findOneOrFetchFromMwl(
            mwlWaifu[APIField.series]?.[APIField.slug] ?? 1
          );
          if (series) {
            series = series[APIField._id];
          }
        }

        logModuleInfo(
          `Caching waifu data for ${mwlWaifu[APIField.name]}`,
          LoggingModule.DB
        );
        const doc = this.create({
          [APIField._id]: mwlWaifu[APIField.id],
          [APIField.mwlSlug]: mwlWaifu[APIField.slug],
          [APIField.mwlCreatorId]: mwlWaifu[APIField.creator].id,
          [APIField.mwlCreatorName]: mwlWaifu[APIField.creator].name,
          [APIField.mwlUrl]: mwlWaifu[APIField.url],
          [APIField.name]: mwlWaifu[APIField.name],
          [APIField.husbando]: mwlWaifu[APIField.husbando],
          [APIField.nsfw]: mwlWaifu[APIField.nsfw],
          [APIField.likes]: mwlWaifu[APIField.likes],
          [APIField.trash]: mwlWaifu[APIField.trash],
          [APIField.description]: mwlWaifu[APIField.description],
          [APIField.originalName]: mwlWaifu[APIField.originalName],
          [APIField.romajiName]: mwlWaifu[APIField.romajiName],
          [APIField.mwlDisplayPictureUrl]: mwlWaifu[APIField.displayPicture],
          [APIField.weight]: mwlWaifu[APIField.weight],
          [APIField.height]: mwlWaifu[APIField.height],
          [APIField.bust]: mwlWaifu[APIField.bust],
          [APIField.hip]: mwlWaifu[APIField.hip],
          [APIField.waist]: mwlWaifu[APIField.waist],
          [APIField.bloodType]: mwlWaifu[APIField.bloodType],
          [APIField.origin]: mwlWaifu[APIField.origin],
          [APIField.age]: mwlWaifu[APIField.age],
          [APIField.birthdayMonth]: mwlWaifu[APIField.birthdayMonth],
          [APIField.birthdayDay]: mwlWaifu[APIField.birthdayDay],
          [APIField.birthdayYear]: mwlWaifu[APIField.birthdayYear],
          [APIField.appearances]: appearances,
          [APIField.series]: series,
        });

        if (updateScores) {
          doc.then(() => {
            this.updateScoresAndTiers();
          });
        }

        return doc;
      }
      return undefined;
    } catch (e) {
      logModuleError(
        `Exception caught while adding new waifu to database: ${e}`,
        LoggingModule.DB
      );
      return undefined;
    }
  }

  public static async getRandom(
    this: ReturnModelType<typeof Waifu>,
    conditions: FilterQuery<DocumentType<Waifu>> = {}
  ): Promise<Waifu | undefined> {
    try {
      const query = await this.leanWaifuQuery(conditions, {}, {}, {}, 0);
      if (query && query.length > 0) {
        const max = query.length;
        const min = 0;
        const randInt = await randomOrg.generateInteger(min, max);
        const waifu = query[randInt];
        if (waifu) return waifu;
      }
      return undefined;
    } catch (e) {
      logModuleError(
        `Exception caught while fetching random waifu from the database: ${e}`,
        LoggingModule.DB
      );
      return undefined;
    }
  }

  public static updateScoresAndTiers(
    this: ReturnModelType<typeof Waifu>
  ): void {
    logModuleInfo('Updating waifu scores and tiers...', LoggingModule.DB);
    const getTier = (pos: number, total: number): Tier => {
      if (pos / total <= 1 / 100) return Tier.S;
      if (pos / total <= 6 / 100) return Tier.A;
      if (pos / total <= 16 / 100) return Tier.B;
      if (pos / total <= 26 / 100) return Tier.C;
      return Tier.D;
    };
    // fetch the waifus with a combined like + trash count of 100+,
    // score = (likes + 1 / trash + 1) * (total # of votes)
    // apply score to our documents
    // sort in descending order
    this.aggregate([
      {
        $project: {
          [APIField.likes]: 1,
          [APIField.trash]: 1,
          total: {
            $add: [`$${APIField.likes}`, `$${APIField.trash}`],
          },
        },
      },
      { $match: { total: { $gt: 100 } } },
      {
        $addFields: {
          [APIField.score]: {
            $multiply: [
              {
                $divide: [
                  { $add: [`$${APIField.likes}`, 1] },
                  { $add: [`$${APIField.trash}`, 1] },
                ],
              },
              { $add: [`$${APIField.likes}`, `$${APIField.trash}`] },
            ],
          },
        },
      },
      { $sort: { [APIField.score]: -1 } },
    ])
      .exec()
      .then((scores: { _id: number; score: number }[]) => {
        scores.forEach((score, i) => {
          this.findByIdAndUpdate(
            score._id,
            {
              $set: {
                [APIField.rank]: i + 1,
                [APIField.score]: score.score,
                [APIField.tier]: getTier(i + 1, scores.length),
              },
            },
            { new: true, strict: false }
          ).exec();
        });
      });
    logModuleInfo('Done updating waifu scores and tiers', LoggingModule.DB);
  }

  public static async leanWaifuQuery(
    this: ReturnModelType<typeof Waifu>,
    conditions: FilterQuery<DocumentType<Waifu>>,
    sort: string | unknown = {},
    projection: unknown | null = {},
    options: QueryFindOptions = {},
    limit = 100,
    cache = true,
    ttl = 30000
  ): Promise<Waifu[] | undefined> {
    const hashedOptions = hash({
      conditions,
      sort,
      projection,
      options,
      limit,
    } as QueryOptions<Waifu>);

    const timeout = (key: string) => {
      logModuleInfo(`${key} deleted from query cache.`, LoggingModule.DB);
      delete cachedQueries[key];
    };

    if (cachedQueries[hashedOptions] && cache) {
      if (timeouts[hashedOptions]) {
        // reset our timer
        clearTimeout(timeouts[hashedOptions]);
        logModuleInfo(
          `Timeout reset for query ${hashedOptions}`,
          LoggingModule.DB
        );
        timeouts[hashedOptions] = setTimeout(() => {
          timeout(hashedOptions);
        }, ttl);
      } else {
        logModuleInfo(
          `Setting timeout for query ${hashedOptions}`,
          LoggingModule.DB
        );
        timeouts[hashedOptions] = setTimeout(() => {
          timeout(hashedOptions);
        }, ttl);
      }
      return cachedQueries[hashedOptions];
    }
    try {
      const query = await this.find(conditions, projection, options)
        .limit(limit)
        .sort(sort)
        .lean()
        .populate(APIField.appearances)
        .populate(APIField.series);
      if (cache) {
        cachedQueries[hashedOptions] = query;
        // set our ttl
        logModuleInfo(
          `Setting timeout for query ${hashedOptions}`,
          LoggingModule.DB
        );
        timeouts[hashedOptions] = setTimeout(() => {
          timeout(hashedOptions);
        }, ttl);
      }

      return query;
    } catch (e) {
      logModuleError(
        `Exception caught when trying to execute a lean Waifu query: ${e}`,
        LoggingModule.DB
      );
      return undefined;
    }
  }
}

export const waifuModel = getModelForClass(Waifu);
