export enum StatusCode {
  Completed,
  Error,
}

interface Result {
  statusCode: StatusCode;
  result?: any;
}

export default Result;
