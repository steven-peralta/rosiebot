"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.waifuApi = void 0;
const config_1 = __importDefault(require("../config"));
const WaifuAPI_1 = __importDefault(require("./WaifuAPI"));
exports.waifuApi = new WaifuAPI_1.default(config_1.default.waifuAPIKey);
//# sourceMappingURL=api.js.map