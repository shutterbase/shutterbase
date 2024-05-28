export enum LOG_LEVEL {
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3,
}

let logLevel = LOG_LEVEL.INFO;

export function setLogLevel(level: LOG_LEVEL): void {
  logLevel = level;
}

export function getLogLevel(): LOG_LEVEL {
  return logLevel;
}

export function getLogLevelString(): string {
  switch (logLevel) {
    case LOG_LEVEL.DEBUG:
      return "debug";
    case LOG_LEVEL.INFO:
      return "info";
    case LOG_LEVEL.WARN:
      return "warn";
    case LOG_LEVEL.ERROR:
      return "error";
  }
}

export function debug(message: string): void {
  log(LOG_LEVEL.DEBUG, message);
}

export function info(message: string): void {
  log(LOG_LEVEL.INFO, message);
}

export function warn(message: string): void {
  log(LOG_LEVEL.WARN, message);
}

export function error(message: string): void {
  log(LOG_LEVEL.ERROR, message);
}

function log(level: LOG_LEVEL, message: string): void {
  if (level >= logLevel) {
    console.log(`[${LOG_LEVEL[level]}] ${message}`);
  }
}
