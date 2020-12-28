import axios, { AxiosInstance } from 'axios';
// import axiosRateLimit from 'axios-rate-limit';
import axiosRateLimit from 'axios-rate-limit';
import { MwlId, MwlSeries, MwlSlug, MwlWaifu } from './types';

export default class WaifuApi {
  private apikey: string;

  private axios: AxiosInstance;

  constructor(apikey: string) {
    this.apikey = apikey;
    this.axios = axiosRateLimit(
      axios.create({
        baseURL: 'https://mywaifulist.moe/api/v1/',
        timeout: 0,
        headers: { apikey },
      }),
      { maxRequests: 1, perMilliseconds: 1000 }
    );
  }

  async getWaifu(id: MwlId | MwlSlug): Promise<MwlWaifu> {
    const { data } = await this.axios.get(`waifu/${id}`);
    return data.data as MwlWaifu;
  }

  // todo: should probably parallelize those axios requests
  getWaifus(ids: MwlId[]): MwlWaifu[] {
    const waifus: MwlWaifu[] = [];
    ids.forEach((id) => {
      this.axios.get(`waifu/${id}`).then((data) => waifus.push(data.data));
    });
    return waifus;
  }

  async getSeries(id: MwlId | MwlSlug): Promise<MwlSeries> {
    const { data } = await this.axios.get(`series/${id}`);
    return data.data as MwlSeries;
  }

  async getRandomWaifu(): Promise<MwlWaifu> {
    const { data } = await this.axios.get('meta/random');
    return data.data as MwlWaifu;
  }
}
