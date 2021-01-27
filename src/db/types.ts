import { DocumentType } from '@typegoose/typegoose';
import { FilterQuery, QueryFindOptions } from 'mongoose';

export interface QueryOptions<T> {
  conditions: FilterQuery<DocumentType<T>>;
  sort: string | unknown;
  projection: unknown | null;
  options: QueryFindOptions;
}
