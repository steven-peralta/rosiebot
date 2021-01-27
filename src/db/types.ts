import { FilterQuery, QueryFindOptions } from 'mongoose';
import { DocumentType } from '@typegoose/typegoose';

export interface QueryOptions<T> {
  conditions: FilterQuery<DocumentType<T>>;
  sort: string | unknown;
  projection: unknown | null;
  options: QueryFindOptions;
}
