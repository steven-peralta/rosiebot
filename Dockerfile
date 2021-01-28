FROM node:12-alpine

WORKDIR /usr/rosiebot

COPY package*.json ./

RUN npm install --global yarn && yarn

COPY . .

CMD ["yarn", "start"]