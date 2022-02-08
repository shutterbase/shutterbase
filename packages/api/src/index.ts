import { ApolloServer } from 'apollo-server-express';
import { ApolloServerPluginDrainHttpServer, ApolloServerPluginLandingPageGraphQLPlayground } from 'apollo-server-core';
import { createServer } from 'http';
import { log, prisma, checkDatabaseConnection } from "./startup";
import { contextBuilder } from "./util";
import schema from "./schema/schemas";
import express = require("express");
import compression = require('compression');

const API_PORT = process.env['API_PORT'] || 5000;

async function main() {

  await checkDatabaseConnection();

  log.info(`Generating express server`)
  const app = express();
  app.use(compression());
  
  log.info(`Creating server`)
  const httpServer = createServer(app);

  log.info(`Generating Apollo server`)
  const server = new ApolloServer({
      schema,
      context: contextBuilder,
      plugins: [
        ApolloServerPluginDrainHttpServer({ httpServer }),
        ApolloServerPluginLandingPageGraphQLPlayground()
      ],
  });
  
  await server.start();
  // @ts-ignore
  server.applyMiddleware({ app });
  await new Promise<void>(resolve => httpServer.listen({ port: API_PORT }, resolve));
  log.info(`GraphQL is now running on port ${API_PORT}`);
}

main()
.catch(e => {
    throw e;
})
.finally(async () => {
    await prisma.$disconnect()
})