import { IResolvers } from '@graphql-tools/utils';

export default function addCustomResolvers(resolvers: IResolvers) {
  if(!resolvers['Query']) {
      resolvers['Query'] = {}
  }

  if(!resolvers['Mutation']) {
      resolvers['Mutation'] = {}
  }
}