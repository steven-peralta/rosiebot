FROM node:12

WORKDIR /usr/rosiebot

COPY package*.json ./

RUN yarn

COPY . .