import axios, { AxiosInstance } from 'axios';
import axiosRateLimit from 'axios-rate-limit';
import { GenerateIntegersRequest, Response, Result } from './types';
import { logError } from '../util/logger';

export default class RandomApi {
  private apiKey: string;

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

  private async makeRequest(body: GenerateIntegersRequest): Promise<Result> {
    const response: Response = (await this.axios.post('/invoke', body)).data;

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
      const result: Result = await this.makeRequest(body);
      return result.random.data as number[];
    } catch (e) {
      logError(e, 'random');
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
      const result: Result = await this.makeRequest(body);
      return result.random.data[0] as number;
    } catch (e) {
      logError(e, 'random');
      throw e;
    }
  }
}
