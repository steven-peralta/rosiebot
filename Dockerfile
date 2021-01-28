FROM node:12-alpine

WORKDIR /usr/rosiebot

COPY package*.json ./

RUN yarn

COPY . .

CMD ["yarn", "start"]