FROM docker.io/node:17

WORKDIR /workdir

COPY package.json ./

RUN yarn

COPY . .

RUN yarn build