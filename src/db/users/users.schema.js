"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const mongoose_1 = require("mongoose");
const users_statics_1 = require("./users.statics");
const users_methods_1 = require("./users.methods");
const UserSchema = new mongoose_1.Schema({
    userId: {
        type: String,
        required: true,
        unique: true,
    },
    coins: {
        type: Number,
        default: 200,
    },
    ownedWaifus: [{ type: Number }],
    dailyLastClaimed: {
        type: Date,
        default: 0,
    },
    dateOfEntry: {
        type: Date,
        default: new Date(),
    },
    lastUpdated: {
        type: Date,
        default: new Date(),
    },
});
UserSchema.statics.findOneOrCreate = users_statics_1.findOneOrCreate;
UserSchema.methods.setLastUpdated = users_methods_1.setLastUpdated;
UserSchema.methods.addWaifu = users_methods_1.addWaifu;
UserSchema.methods.setCoins = users_methods_1.setCoins;
UserSchema.methods.setDailyClaimed = users_methods_1.setDailyClaimed;
exports.default = UserSchema;
