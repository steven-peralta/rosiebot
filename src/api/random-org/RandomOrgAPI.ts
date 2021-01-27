import axios, { AxiosInstance } from 'axios';
import axiosRateLimit from 'axios-rate-limit';
import config from 'rosiebot/src/config';
import {
  LoggingModule,
  logModuleError,
  logModuleInfo,
} from 'rosiebot/src/util/logger';
import {
  GenerateIntegersRequest,
  RandomOrgResponse,
  RandomOrgResult,
} from 'rosiebot/src/api/random-org/types';

class RandomOrgAPI {
  private readonly apiKey: string;

  private axios: AxiosInstance;

  constructor(apiKey: string) {
    this.apiKey = apiKey;
    this.axios = axiosRateLimit(
      axios.create({
        baseURL: 'https://api.random.org/json-rpc/2',
      }),
      { maxRPS: 10 }
    );
  }

  private async makeRequest(
    body: GenerateIntegersRequest
  ): Promise<RandomOrgResult> {
    logModuleInfo(`POST: ${this.axios.defaults.baseURL}/invoke`);
    const response: RandomOrgResponse = (await this.axios.post('/invoke', body))
      .data;

    if (response.error) {
      throw new Error(`${response.error.code}: ${response.error.message}`);
    } else if (response.result) {
      return response.result;
    } else {
      throw new Error('No result or error object in response.');
    }
  }

  async generateIntegers(
    min: number,
    max: number,
    n: number,
    replacement = true,
    base = 10
  ): Promise<number[]> {
    const body: GenerateIntegersRequest = {
      jsonrpc: '2.0',
      id: 52,
      method: 'generateIntegers',
      params: {
        apiKey: this.apiKey,
        n,
        min,
        max,
        replacement,
        base,
      },
    };

    try {
      const start = Date.now();
      const result: RandomOrgResult = await this.makeRequest(body);
      logModuleInfo(
        `Generated ${
          result.random.data.length
        } random ints (max: ${max} min: ${min}) in ${Date.now() - start} ms`
      );
      return result.random.data as number[];
    } catch (e) {
      logModuleError(
        `Exception caught when trying to generate random integers: ${e}`,
        LoggingModule.RandomOrg
      );
      throw e;
    }
  }

  async generateInteger(min: number, max: number, base = 10): Promise<number> {
    const body: GenerateIntegersRequest = {
      jsonrpc: '2.0',
      id: 52,
      method: 'generateIntegers',
      params: {
        apiKey: this.apiKey,
        n: 1,
        min,
        max,
        base,
      },
    };

    try {
      const start = Date.now();
      const result: RandomOrgResult = await this.makeRequest(body);
      logModuleInfo(
        `Generated random int ${
          result.random.data[0]
        } (max: ${max} min: ${min}) in ${Date.now() - start} ms`,
        LoggingModule.RandomOrg
      );
      return result.random.data[0] as number;
    } catch (e) {
      logModuleError(
        `Exception caught when trying to generate random integers: ${e}`,
        LoggingModule.RandomOrg
      );
      throw e;
    }
  }
}

const randomOrgAPIInstance = new RandomOrgAPI(config.randomOrgApiKey);

export default randomOrgAPIInstance;
