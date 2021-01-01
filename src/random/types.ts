export interface Base {
  jsonrpc: string;
  id: number;
}

export interface Request extends Base {
  method: string;
}

export interface Error {
  code: number;
  message: string;
  data: string[];
}

export interface Result {
  random: {
    data: number[] | string[];
    completionTime: string;
  };
  bitsUsed: number;
  bitsLeft: number;
  requestsLeft: number;
  advisoryDelay: number;
}

export interface Response extends Base {
  error?: Error;
  result?: Result;
}

export interface GenerateIntegersRequest extends Request {
  params: {
    apiKey: string;
    n: number;
    min: number;
    max: number;
    replacement?: boolean;
    base?: number;
  };
}
