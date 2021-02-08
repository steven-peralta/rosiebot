// eslint-disable-next-line @typescript-eslint/no-var-requires
const tsconfigPaths = require('tsconfig-paths');
// eslint-disable-next-line @typescript-eslint/no-var-requires
const tsconfig = require('../tsconfig.json');

const baseUrl = './dist';
tsconfigPaths.register({
  baseUrl,
  paths: tsconfig.compilerOptions.paths,
});
