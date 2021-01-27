import { CommandMetadata } from '@commands/types';
import { logModuleError, logModuleInfo, logModuleWarning } from '@util/logger';
import { StatusCode } from '@util/enums';

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
