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
exports.findOneOrCreate = void 0;
function findOneOrCreate(waifu) {
    var _a, _b, _c;
    return __awaiter(this, void 0, void 0, function* () {
        const record = yield this.findOne({ mwlId: waifu.id });
        if (record) {
            return record;
        }
        else {
            const appearsIn = (_b = (_a = waifu.appearances) === null || _a === void 0 ? void 0 : _a.reduce((acc, val) => {
                acc.push(val.name);
                return acc;
            }, [])) !== null && _b !== void 0 ? _b : [];
            return this.create({
                mwlId: waifu.id,
                name: waifu.name,
                url: waifu.url,
                originalName: waifu.original_name,
                displayPictureURL: waifu.display_picture,
                description: waifu.description,
                weight: waifu.weight,
                height: waifu.height,
                bust: waifu.bust,
                hip: waifu.hip,
                waist: waifu.waist,
                bloodType: waifu.blood_type,
                origin: waifu.origin,
                age: waifu.age,
                birthdayMonth: waifu.birthday_month,
                birthdayDay: waifu.birthday_day,
                birthdayYear: waifu.birthday_year,
                husbando: waifu.husbando,
                appearsIn,
                series: (_c = waifu.series) === null || _c === void 0 ? void 0 : _c.name,
            });
        }
    });
}
exports.findOneOrCreate = findOneOrCreate;
