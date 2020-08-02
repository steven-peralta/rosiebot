"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.setCoins = exports.addWaifu = exports.setDailyClaimed = exports.setLastUpdated = void 0;
function setLastUpdated() {
    return __awaiter(this, void 0, void 0, function* () {
        const now = new Date();
        if (!this.lastUpdated || this.lastUpdated < now) {
            this.lastUpdated = now;
            yield this.save();
        }
    });
}
exports.setLastUpdated = setLastUpdated;
function setDailyClaimed() {
    return __awaiter(this, void 0, void 0, function* () {
        const now = new Date();
        if (!this.dailyLastClaimed || this.dailyLastClaimed < now) {
            this.dailyLastClaimed = now;
            yield this.save();
        }
    });
}
exports.setDailyClaimed = setDailyClaimed;
function addWaifu(waifuId) {
    return __awaiter(this, void 0, void 0, function* () {
        if (!this.ownedWaifus) {
            this.ownedWaifus = [waifuId];
            yield this.setLastUpdated();
            yield this.save();
        }
        else {
            if (this.ownedWaifus.includes(waifuId))
                return;
            this.ownedWaifus.push(waifuId);
            yield this.setLastUpdated();
            yield this.save();
        }
    });
}
exports.addWaifu = addWaifu;
function setCoins(coins) {
    return __awaiter(this, void 0, void 0, function* () {
        this.coins = coins;
        yield this.setLastUpdated();
        yield this.save();
    });
}
exports.setCoins = setCoins;
