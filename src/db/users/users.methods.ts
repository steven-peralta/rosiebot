import { IUserDocument } from "./users.types";

export async function setLastUpdated(this: IUserDocument): Promise<void> {
  const now = new Date();
  if (!this.lastUpdated || this.lastUpdated < now) {
    this.lastUpdated = now;
    await this.save();
  }
}

export async function setDailyClaimed(this: IUserDocument): Promise<void> {
  const now = new Date();
  if (!this.dailyLastClaimed || this.dailyLastClaimed < now) {
    this.dailyLastClaimed = now;
    await this.save();
  }
}

export async function addWaifu(
  this: IUserDocument,
  waifuId: number
): Promise<void> {
  if (!this.ownedWaifus) {
    this.ownedWaifus = [waifuId];
    await this.setLastUpdated();
    await this.save();
  } else {
    if (this.ownedWaifus.includes(waifuId)) return;
    this.ownedWaifus.push(waifuId);
    await this.setLastUpdated();
    await this.save();
  }
}

export async function setCoins(
  this: IUserDocument,
  coins: number
): Promise<void> {
  this.coins = coins;
  await this.setLastUpdated();
  await this.save();
}
