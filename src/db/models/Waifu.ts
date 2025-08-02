import {
  DocumentType,
  getModelForClass,
  index,
  pre,
  prop,
  Ref,
  ReturnModelType,
} from '@typegoose/typegoose';
import { Base } from '@typegoose/typegoose/lib/defaultClasses';
import { FilterQuery, QueryFindOptions } from 'mongoose';
import APIField from '$util/APIField';
import Series, { seriesModel } from '$db/models/Series';
import mwl from '$api/mwl/mwl';
import { LoggingModule, logModuleError, logModuleInfo } from '$util/logger';
import { Tier } from '$util/enums';

function setLastUpdated(this: DocumentType<Waifu>) {
  const now = new Date();
  if (!this[APIField.updated] || this[APIField.updated] < now) {
    this[APIField.updated] = now;
  }
}

@pre<Waifu>('save', setLastUpdated)
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

  @prop({ default: new Date() })
  [APIField.created]!: Date;

  @prop({ default: new Date() })
  [APIField.updated]!: Date;

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

  public static async updateFromMWL(
    this: ReturnModelType<typeof Waifu>,
    mwlId: number | string,
    updateScores = false
  ): Promise<Waifu | undefined> {
    try {
      const mwlWaifu = await mwl.getWaifu(mwlId);

      if (mwlWaifu) {
        const appearances = (
          await Promise.all(
            (mwlWaifu[APIField.appearances] ?? [])
              .map((appearance) => appearance[APIField.slug])
              .map((slug) => seriesModel.updateFromMWL(slug))
          )
        ).map((doc) => doc?.[APIField._id] ?? 0); // if the series doc some how ends up being undefined, use a predetermined id value

        let series;
        if (
          mwlWaifu[APIField.series] &&
          mwlWaifu[APIField.series]?.[APIField.slug]
        ) {
          series = await seriesModel.updateFromMWL(
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

        const updateOpts = {
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
          [APIField.updated]: new Date(),
        };

        let record =
          typeof mwlId === 'number'
            ? await this.findById(mwlId)
            : await this.findOne({ [APIField.mwlSlug]: mwlId });

        if (record) {
          await record.updateOne(updateOpts);
        } else {
          // FIXME need to update the typegoose types to allow this
          // eslint-disable-next-line @typescript-eslint/ban-ts-comment
          // @ts-ignore
          record = await this.create({
            ...updateOpts,
            [APIField.created]: new Date(),
          });
        }

        if (updateScores) {
          await this.updateScoresAndTiers();
        }

        return record;
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

  public static async random(
    this: ReturnModelType<typeof Waifu>,
    conditions: FilterQuery<DocumentType<Waifu>> = {}
  ): Promise<Waifu | undefined> {
    try {
      const query = await this.aggregate([
        { $match: conditions },
        {
          $lookup: {
            from: 'series',
            localField: APIField.series,
            foreignField: APIField._id,
            as: APIField.series,
          },
        },
        {
          $lookup: {
            from: 'series',
            localField: APIField.appearances,
            foreignField: APIField._id,
            as: APIField.appearances,
          },
        },
        { $sample: { size: 1 } },
      ]);
      if (query && query.length > 0) {
        return query[0] as Waifu;
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

  public static async updateScoresAndTiers(
    this: ReturnModelType<typeof Waifu>
  ): Promise<void> {
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
    const scores: { _id: number; score: number }[] = await this.aggregate([
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
    ]).exec();

    const promises: Promise<DocumentType<Waifu> | null>[] = [];
    scores.forEach((score, i) => {
      promises.push(
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
        ).exec()
      );
    });
    await Promise.all(promises);
    logModuleInfo('Done updating waifu scores and tiers', LoggingModule.DB);
  }

  public static async leanFind(
    this: ReturnModelType<typeof Waifu>,
    conditions: FilterQuery<DocumentType<Waifu>>,
    sort: string | unknown = {},
    projection: unknown | null = {},
    options: QueryFindOptions = {},
    limit = 100
  ): Promise<Waifu[] | undefined> {
    try {
      return await this.find(conditions, projection, options)
        .limit(limit)
        .sort(sort)
        .lean()
        .populate(APIField.appearances)
        .populate(APIField.series);
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
