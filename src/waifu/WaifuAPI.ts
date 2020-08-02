import axios, { AxiosInstance } from "axios";
import { Waifu } from "./api";
import axiosRateLimit from "axios-rate-limit";

export default class WaifuAPI {
  private apikey: string;
  private axios: AxiosInstance;

  constructor(apikey: string) {
    this.apikey = apikey;
    this.axios = axiosRateLimit(
      axios.create({
        baseURL: "https://mywaifulist.moe/api/v1/",
        timeout: 1000,
        headers: { apikey },
      }),
      { maxRequests: 1, perMilliseconds: 1000 }
    );
  }

  async getWaifu(id: number): Promise<Waifu> {
    const { data } = await this.axios.get(`waifu/${id}`);
    return data.data as Waifu;
  }

  async getWaifus(ids: string[]): Promise<Waifu[]> {
    const waifus: Waifu[] = [];
    for (const id of ids) {
      const { data } = await this.axios.get(`waifu/${id}`);
      waifus.push(data.data);
    }
    return waifus;
  }

  async getRandomWaifu(): Promise<Waifu> {
    const { data } = await this.axios.get("meta/random");
    return data.data as Waifu;
  }
}
