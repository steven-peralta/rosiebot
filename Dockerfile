FROM node:22

WORKDIR /usr/rosiebot

COPY package*.json ./

RUN npm install

COPY . .
