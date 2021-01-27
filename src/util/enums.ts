export enum Command {
  wcoins = 'wcoins',
  wsearch = 'wsearch',
  wowned = 'wowned',
  wrandom = 'wrandom',
  wotd = 'wotd',
  wroll = 'wroll',
  wtrade = 'wtrade',
  ssearch = 'ssearch',
  wdaily = 'wdaily',
}

export enum StatusCode {
  Success,
  Error,
  NoData,
  IncorrectArgs,
  InsufficientArgs,
  NoPermission,
  InsufficentCoins,
  UserOwnsNoWaifus,
}

export const ErrorMessage: Record<StatusCode, string> = {
  [StatusCode.Success]: 'Command executed successfully.',
  [StatusCode.Error]: 'An unexpected error occurred.',
  [StatusCode.NoData]: 'No data was found.',
  [StatusCode.IncorrectArgs]: 'The command arguments provided are incorrect.',
  [StatusCode.InsufficientArgs]:
    'An insufficient number of arguments were provided.',
  [StatusCode.NoPermission]: "You don't have permission to run that command",
  [StatusCode.InsufficentCoins]: "You don't have enough coins!",
  [StatusCode.UserOwnsNoWaifus]: "Specified user doesn't own any waifus.",
};
