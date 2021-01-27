module.exports = {
  env: {
    browser: false,
    es2020: true,
  },
  extends: ['airbnb-typescript-prettier'],
  parserOptions: {
    sourceType: "module",
    tsconfigRootDir: __dirname,
    project: './tsconfig.json',
    ecmaVersion: 11,
  },
  settings: {
    "import/resolver": {
      "node": {
        "moduleDirectory": ["node_modules", "src/"]
      },
      "typescript": {
      }
    },
  },
  rules: {
    'no-use-before-define': 0,
    'no-underscore-dangle': 0,
    '@typescript-eslint/no-unused-vars': ['error', { "argsIgnorePattern": "^_" }]
  },
};
