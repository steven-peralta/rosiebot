import axios, { AxiosInstance } from 'axios';
import { MwlSeries, MwlWaifu } from './types';

export default class WaifuApi {
  private apikey: string;

  private axios: AxiosInstance;

  constructor(apikey: string) {
    this.apikey = apikey;
    this.axios = axios.create({
      baseURL: 'https://mywaifulist.moe/api/v1/',
      timeout: 0,
      headers: { apikey },
    });
  }

  async getWaifu(id: number | string): Promise<MwlWaifu> {
    const { data } = await this.axios.get(`waifu/${id}`);
    return data.data as MwlWaifu;
  }

  async getSeries(id: number | string): Promise<MwlSeries> {
    const { data } = await this.axios.get(`series/${id}`);
    return data.data as MwlSeries;
  }
}
