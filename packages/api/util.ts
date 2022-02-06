import {PrismaClient} from '@prisma/client';
import 'dotenv/config'

const PRISMA_LOG_LEVEL = process.env.PRISMA_LOG_LEVEL || ['info', 'warn', 'error'];

const primsa = new PrismaClient({
    log: PRISMA_LOG_LEVEL,
});