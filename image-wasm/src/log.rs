//! Tiny leveled logger. Replaces the old `static mut` + custom `console.log`
//! binding with an `AtomicU8` threshold (no `unsafe`) and `web_sys::console`.
//!
//! The actual emit is cfg-gated: on `wasm32` it goes to the browser console, on
//! the host (`cargo test`) it goes to stderr — so pure-logic functions can log
//! freely without breaking native unit tests.

use std::sync::atomic::{AtomicU8, Ordering};

const DEBUG: u8 = 0;
const INFO: u8 = 1;
const WARN: u8 = 2;
const ERROR: u8 = 3;

static LEVEL: AtomicU8 = AtomicU8::new(INFO);

/// Set the minimum level to emit. Unknown strings fall back to `info`.
pub fn set_level(level: &str) {
    let value = match level {
        "debug" => DEBUG,
        "info" => INFO,
        "warn" => WARN,
        "error" => ERROR,
        _ => INFO,
    };
    LEVEL.store(value, Ordering::Relaxed);
}

fn emit(line: &str) {
    #[cfg(target_arch = "wasm32")]
    web_sys::console::log_1(&wasm_bindgen::JsValue::from_str(line));
    #[cfg(not(target_arch = "wasm32"))]
    eprintln!("{line}");
}

fn log(level: u8, tag: &str, message: &str) {
    if level >= LEVEL.load(Ordering::Relaxed) {
        emit(&format!("[{tag}] {message}"));
    }
}

// The logger keeps a complete level set; not every level is called today.
pub fn debug(message: &str) {
    log(DEBUG, "DEBUG", message);
}
#[allow(dead_code)]
pub fn info(message: &str) {
    log(INFO, "INFO", message);
}
#[allow(dead_code)]
pub fn warn(message: &str) {
    log(WARN, "WARN", message);
}
#[allow(dead_code)]
pub fn error(message: &str) {
    log(ERROR, "ERROR", message);
}
