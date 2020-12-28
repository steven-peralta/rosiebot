module.exports = {
  env: {
    browser: true,
    es2020: true,
  },
  extends: ['airbnb-typescript-prettier'],
  parser: '@typescript-eslint/parser',
  parserOptions: {
    sourceType: "module",
    tsconfigRootDir: __dirname,
    project: './tsconfig.json',
    ecmaVersion: 11,
  },
  plugins: ['@typescript-eslint'],
  rules: {
    'no-use-before-define': 0,
    'no-underscore-dangle': 0,
    'no-bitwise': ["error", { "int32Hint": true }]
  },
};
