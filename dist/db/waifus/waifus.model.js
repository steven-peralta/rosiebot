"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.WaifuModel = void 0;
const waifus_schema_1 = __importDefault(require("./waifus.schema"));
const mongoose_1 = require("mongoose");
exports.WaifuModel = mongoose_1.model('waifu', waifus_schema_1.default);
//# sourceMappingURL=waifus.model.js.map