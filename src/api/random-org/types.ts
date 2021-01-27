export interface RandomOrgBase {
  jsonrpc: string;
  id: number;
}

export interface RandomOrgRequest extends RandomOrgBase {
  method: string;
}

export interface RandomOrgError {
  code: number;
  message: string;
  data: string[];
}

export interface RandomOrgResult {
  random: {
    data: number[] | string[];
    completionTime: string;
  };
  bitsUsed: number;
  bitsLeft: number;
  requestsLeft: number;
  advisoryDelay: number;
}

export interface RandomOrgResponse extends RandomOrgBase {
  error?: RandomOrgError;
  result?: RandomOrgResult;
}

export interface GenerateIntegersRequest extends RandomOrgRequest {
  params: {
    apiKey: string;
    n: number;
    min: number;
    max: number;
    replacement?: boolean;
    base?: number;
  };
}
