import {
  logModuleError,
  logModuleInfo,
  logModuleWarning,
} from 'rosiebot/src/util/logger';
import { CommandMetadata } from 'rosiebot/src/commands/types';
import { StatusCode } from 'rosiebot/src/util/enums';

export const logCommandException = (
  exception: Error,
  metadata: CommandMetadata
): void => {
  logModuleError(
    `Exception caught while executing command ${metadata.name}: ${exception}`
  );
};

export const logCommandStatus = (
  metadata: CommandMetadata,
  statusCode: StatusCode
): void => {
  if (statusCode !== StatusCode.Success) {
    logModuleWarning(
      `${metadata.name} returned to a user with status code ${statusCode}`
    );
  } else {
    logModuleInfo(
      `${metadata.name} returned to a user with a successful status code.`
    );
  }
};
