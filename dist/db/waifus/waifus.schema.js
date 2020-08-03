"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const mongoose_1 = require("mongoose");
const waifus_statics_1 = require("./waifus.statics");
const WaifuSchema = new mongoose_1.Schema({
    mwlId: {
        type: Number,
        required: true,
    },
    name: {
        type: String,
        required: true,
    },
    url: {
        type: String,
        required: true,
    },
    originalName: String,
    displayPictureURL: String,
    description: String,
    weight: String,
    height: String,
    bust: String,
    hip: String,
    waist: String,
    bloodType: String,
    origin: String,
    age: Number,
    birthdayMonth: String,
    birthdayDay: Number,
    birthdayYear: Number,
    husbando: Boolean,
    series: String,
    appearsIn: [String],
});
WaifuSchema.statics.findOneOrCreate = waifus_statics_1.findOneOrCreate;
exports.default = WaifuSchema;
//# sourceMappingURL=waifus.schema.js.map