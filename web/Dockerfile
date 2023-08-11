FROM node:18-alpine3.18 as build

RUN npm install --location=global pnpm

WORKDIR /usr/src
COPY package.json /usr/src/package.json
COPY pnpm-lock.yaml /usr/src/pnpm-lock.yaml
RUN pnpm install --shamefully-hoist

COPY . /usr/src

RUN pnpm run build


FROM alpine:3.18
RUN apk add --no-cache dumb-init nodejs
USER 1000

WORKDIR /usr/app
COPY --chown=1000:1000 --from=build /usr/src/.output ./

EXPOSE 8080
ENV HOST=0.0.0.0 PORT=8080 NODE_ENV=production
CMD ["dumb-init","node", "--trace-warnings","/usr/app/server/index.mjs"]