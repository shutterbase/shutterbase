import { AuthenticationError } from 'apollo-server';
import { IncomingMessage } from 'http';
import { prisma as startupPrisma } from './startup'
export const prisma = startupPrisma;

import { log as startupLog } from './startup'
export const log = startupLog;

type Req = {  req: IncomingMessage }
export async function contextBuilder({req}: Req): Promise<ContextType> {
  if(req.headers["authorization"] === null || req.headers["authorization"] === undefined) {
    return { auth: false, user: null };
  }
  else {
    return { auth: false, user: null };
  }
}

type UserGroupType = {
  id: string,
  key: string,
  name: string
}

type UserType = {
  id: string,
  email: string,
  username: string,
  firstName: string,
  lastName: string,
  groups: Array<UserGroupType>
  laboratories: Array<any>
}

export type ContextType = {
  auth: Boolean,
  user: UserType | null
}

export function authBlock(context: ContextType) {
  if (!context.auth) {
      throw new AuthenticationError('request not authenticated');
  }
}