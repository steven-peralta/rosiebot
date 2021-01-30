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
  SeriesNotFound,
  WaifuNotFound,
  StudioNotFound,
  UserNotFound,
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
  [StatusCode.WaifuNotFound]: 'Waifu was not found.',
  [StatusCode.SeriesNotFound]: 'Series was not found.',
  [StatusCode.StudioNotFound]: 'Studio was not found.',
  [StatusCode.UserNotFound]:
    "User was not found. (Are you sure you @'d them correctly?)",
};

export enum Tier {
  S = 5, // top 1% of waifus
  A = 4, // next 6% of waifus
  B = 3, // next 16% of waifus
  C = 2, // next 26% of waifus
  D = 1, // next 51% of waifus
}

export enum Button {
  Forward = '▶️',
  Backward = '◀️',
  FastForward = '⏭',
  Rewind = '⏮',
  Jump = '⤴️',
  Checkmark = '✅',
  Cancel = '🚫',
  MoneyBag = '💰',
}
