import 'graphql-import-node';
import resolvers from '../controller/resolvers';
import addCustomResolvers from '../controller/custom-resolvers';
import {buildSubgraphSchema} from "@apollo/federation";
import * as fs from "fs";
const { mergeTypeDefs } = require('@graphql-tools/merge');
import { gql } from 'apollo-server-express';
const path = require("path");

const types = gql(fs.readFileSync(path.join(__dirname, 'types.graphql'), 'utf8'))
const inputs = gql(fs.readFileSync(path.join(__dirname, 'inputs.graphql'), 'utf8'))
const operations = gql(fs.readFileSync(path.join(__dirname, 'operations.graphql'), 'utf8'))

let typeDefs = mergeTypeDefs([types, inputs, operations]);

addCustomResolvers(resolvers)
//@ts-ignore
const schema = buildSubgraphSchema([{ typeDefs, resolvers }]);
export default schema;