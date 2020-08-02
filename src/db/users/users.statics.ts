import { IUserDocument, IUserModel } from "./users.types";
import { Snowflake } from "discord.js";

export async function findOneOrCreate(
  this: IUserModel,
  { userId }: { userId: Snowflake }
): Promise<IUserDocument> {
  const record = await this.findOne({ userId });
  if (record) {
    return record;
  } else {
    return this.create({ userId });
  }
}
