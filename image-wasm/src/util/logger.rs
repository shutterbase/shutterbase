use crate::util::js;

pub enum LogLevel {
    DEBUG,
    INFO,
    WARN,
    ERROR,
}

static mut log_level: LogLevel = LogLevel::INFO;

pub fn debug(message: &str) {
    log(LogLevel::DEBUG, message);
}

pub fn info(message: &str) {
    log(LogLevel::INFO, message);
}

pub fn warn(message: &str) {
    log(LogLevel::WARN, message);
}

pub fn error(message: &str) {
    log(LogLevel::ERROR, message);
}

fn log(level: LogLevel, message: &str) {
    let level_str = match level {
        LogLevel::DEBUG => "DEBUG",
        LogLevel::INFO => "INFO",
        LogLevel::WARN => "WARN",
        LogLevel::ERROR => "ERROR",
    };

    let log_level_threshold = unsafe {
        match log_level {
            LogLevel::DEBUG => 0,
            LogLevel::INFO => 1,
            LogLevel::WARN => 2,
            LogLevel::ERROR => 3,
        }
    };
    let level = level as i32;

    if level < log_level_threshold {
        return;
    }

    let log_message = format!("[{}] {}", level_str, message);
    js::log(&log_message);
}

pub fn set_log_level(level: LogLevel) {
    unsafe { log_level = level };
}

pub fn set_log_level_string(level: String) {
    match level.as_str() {
        "debug" => set_log_level(LogLevel::DEBUG),
        "info" => set_log_level(LogLevel::INFO),
        "warn" => set_log_level(LogLevel::WARN),
        "error" => set_log_level(LogLevel::ERROR),
        _ => set_log_level(LogLevel::INFO),
    }
}
