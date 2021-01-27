import { QueryOptions } from 'rosiebot/src/db/types';
import { FilterQuery } from 'mongoose';
import { DocumentType } from '@typegoose/typegoose';
import APIField from 'rosiebot/src/util/APIField';
import { Waifu } from 'rosiebot/src/db/models/Waifu';

export const parseWaifuSearchArgs = (args: string[]): QueryOptions<Waifu> => {
  const conditions: FilterQuery<DocumentType<Waifu>> = {};
  const projection: Record<string, unknown> = {};
  const sort: Record<string, number | unknown> = {};
  const text: string[] = [];
  args.forEach((arg) => {
    if (arg.match(':')) {
      const split = arg.split(':');
      let field = split[0];
      let value = split[1];
      let operator;
      if (field === 'sortby') {
        let order;
        switch (value.charAt(0)) {
          case '+':
            order = 1;
            value = value.substring(1);
            break;
          case '-':
            order = -1;
            value = value.substring(1);
            break;
          default:
            order = 1;
        }
        if (value === 'id') value = APIField._id;
        sort[value] = order;
        conditions[value] = { $ne: undefined };
      } else {
        switch (value.charAt(0)) {
          case '<':
            operator = '$lt';
            value = value.substring(1);
            break;
          case '<=':
            operator = '$lte';
            value = value.substring(1);
            break;
          case '>':
            operator = '$gt';
            value = value.substring(1);
            break;
          case '>=':
            operator = '$gte';
            value = value.substring(1);
            break;
          default:
            operator = '$eq';
        }
        if (field === 'id') field = APIField._id;
        conditions[field] = { [operator]: value };
      }
    } else {
      text.push(arg);
    }
  });
  if (text.length > 0) {
    conditions.$text = { $search: text.join(' ') };

    // if we have no sort set, then sort by Mongo's text score algorithm
    if (Object.keys(sort).length === 0) {
      sort.score = { $meta: 'textScore' };
      projection.score = { $meta: 'textScore' };
    }
  }
  return {
    conditions,
    sort,
    projection,
    options: {},
  };
};
