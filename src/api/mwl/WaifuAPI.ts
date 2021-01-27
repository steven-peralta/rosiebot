import axios, { AxiosInstance } from 'axios';
import axiosRateLimit from 'axios-rate-limit';
import { MwlSeries, MwlWaifu } from 'rosiebot/src/api/mwl/types';
import config from 'rosiebot/src/config';
import {
  LoggingModule,
  logModuleError,
  logModuleInfo,
  logModuleWarning,
} from 'rosiebot/src/util/logger';

class WaifuAPI {
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
      { maxRPS: 1 }
    );
  }

  async getWaifu(id: number | string): Promise<MwlWaifu | undefined> {
    try {
      const start = Date.now();
      logModuleInfo(
        `GET: ${this.axios.defaults.baseURL}/waifu/${id}`,
        LoggingModule.MWL
      );
      const { data, status } = await this.axios.get(`waifu/${id}`);
      logModuleInfo(
        `Fetched waifu ${id} from MWL in ${Date.now() - start} ms`,
        LoggingModule.MWL
      );
      if (status === 200) {
        return data.data as MwlWaifu;
      }
      logModuleWarning(
        `Received ${status} status from ${this.axios.defaults.baseURL}/waifu/${id}`,
        LoggingModule.MWL
      );
      return undefined;
    } catch (e) {
      logModuleError(
        `Exception caught while trying to fetch Waifu data: ${e}`,
        LoggingModule.MWL
      );
      return undefined;
    }
  }

  async getSeries(id: number | string): Promise<MwlSeries | undefined> {
    try {
      const start = Date.now();
      logModuleInfo(
        `GET: ${this.axios.defaults.baseURL}/series/${id}`,
        LoggingModule.MWL
      );
      const { data, status } = await this.axios.get(`series/${id}`);
      logModuleInfo(
        `Fetched series ${id} from MWL in ${Date.now() - start} ms`,
        LoggingModule.MWL
      );
      if (status === 200) {
        return data.data as MwlSeries;
      }
      logModuleWarning(
        `Received ${status} status from ${this.axios.defaults.baseURL}/waifu/${id}`,
        LoggingModule.MWL
      );
      return undefined;
    } catch (e) {
      logModuleError(
        `Exception caught while trying to fetch Series data: ${e}`,
        LoggingModule.MWL
      );
      return undefined;
    }
  }
}

const waifuAPIInstance = new WaifuAPI(config.waifuApiKey);

export default waifuAPIInstance;
