import { Prisma, PrismaClient } from '@prisma/client';
import 'dotenv/config'
import winston = require('winston');
const { format } = require('logform');

// Prisma log levels
const DEFAULT_PRISMA_LOG_LEVELS: Prisma.LogLevel[] = ['info', 'warn', 'error'];
const PRISMA_LOG_LEVELS = parsePrismaLogLevels(process.env['PRISMA_LOG_LEVEL']);
console.log(`using prisma log levels '${PRISMA_LOG_LEVELS}'`);

// winston log levels
const AVAILABLE_LOG_LEVELS = ['error','warn','info','http','verbose','debug','silly'];
const DEFAULT_LOG_LEVEL: string = 'info';
export const LOG_LEVEL = parseLogLevel(process.env['LOG_LEVEL']);
console.log(`using log level '${LOG_LEVEL}'`);

const LOG_FORMAT = format.combine(
  format.colorize(),
  format.timestamp(),
  format.align(),
  format.printf((info: any) => `${info.timestamp} ${info.level}: ${info.message}`)
);

function failStartup() {
  console.error('Failed to start. See reason above. Exiting.');
  process.exit(1);
}

function parsePrismaLogLevels(s: string|undefined): Prisma.LogLevel[] {
  const AVAILABLE_PRISMA_LOG_LEVELS = ['query', 'warn', 'info', 'error'];
  let levels: Prisma.LogLevel[] = [];
  if (s) {
    s.split(',').forEach(level => {
      level = level.trim();
      if (AVAILABLE_PRISMA_LOG_LEVELS.includes(level)) {
        levels.push(level as Prisma.LogLevel);
      } else {
        console.error(`Unknown log level '${level}'`);
        failStartup();
        throw new Error('Unknown log level');
      }
    });
  }
  else {
    return DEFAULT_PRISMA_LOG_LEVELS;
  }
  return levels;
}

function parseLogLevel(level: string|undefined): string {
  if(level) {
    if(AVAILABLE_LOG_LEVELS.includes(level)) {
      return level;
    } else {
      console.warn(`Unknown log level '${level}'`);
      failStartup();
      throw new Error('Unknown log level');
    }
  }
  else {
    return DEFAULT_LOG_LEVEL;
  }
}

export const log = winston.createLogger({
  level: LOG_LEVEL,
  format: LOG_FORMAT,
  transports: [new winston.transports.Console()]
});

log.info(`Hello Logger`)

export const prisma = new PrismaClient({
  log: PRISMA_LOG_LEVELS,
});

if (LOG_LEVEL === 'debug') {
  prisma.$use(async (params, next) => {
    const before = Date.now()
    const result = await next(params)
    const after = Date.now()
    log.debug(`Query ${params.model}.${params.action} took ${after - before}ms`)
    return result
  })
}

export async function checkDatabaseConnection() {
  try {
    const pictureCount = await prisma.picture.count();
    log.info(`Database connection successful. ${pictureCount} pictures in database.`);
  }
  catch(e) {
    log.error(`Could not connect to database. Exiting.`);
    failStartup();
  }
}