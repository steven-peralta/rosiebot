import { CommandMetadata } from './types';
import { logError } from '../util/logger';

const logCommandException = (
  exception: Error,
  metadata: CommandMetadata
): void => {
  logError(
    `Exception caught while executing command ${metadata.name}: ${exception}`
  );
};

export default logCommandException;
