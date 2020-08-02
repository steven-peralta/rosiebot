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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const axios_1 = __importDefault(require("axios"));
const axios_rate_limit_1 = __importDefault(require("axios-rate-limit"));
class WaifuAPI {
    constructor(apikey) {
        this.apikey = apikey;
        this.axios = axios_rate_limit_1.default(axios_1.default.create({
            baseURL: "https://mywaifulist.moe/api/v1/",
            timeout: 1000,
            headers: { apikey },
        }), { maxRequests: 1, perMilliseconds: 1000 });
    }
    getWaifu(id) {
        return __awaiter(this, void 0, void 0, function* () {
            const { data } = yield this.axios.get(`waifu/${id}`);
            return data.data;
        });
    }
    getWaifus(ids) {
        return __awaiter(this, void 0, void 0, function* () {
            const waifus = [];
            for (const id of ids) {
                const { data } = yield this.axios.get(`waifu/${id}`);
                waifus.push(data.data);
            }
            return waifus;
        });
    }
    getRandomWaifu() {
        return __awaiter(this, void 0, void 0, function* () {
            const { data } = yield this.axios.get("meta/random");
            return data.data;
        });
    }
}
exports.default = WaifuAPI;
