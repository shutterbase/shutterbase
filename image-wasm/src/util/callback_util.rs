use wasm_bindgen::prelude::*;

pub enum CallbackStatus {
    RESIZING,
    RESIZED,
    UPLOADING,
    UPLOADED,
    ERROR,
}

impl CallbackStatus {
    fn as_str(&self) -> &'static str {
        match self {
            CallbackStatus::RESIZING => "resizing",
            CallbackStatus::RESIZED => "resized",
            CallbackStatus::UPLOADING => "uploading",
            CallbackStatus::UPLOADED => "uploaded",
            CallbackStatus::ERROR => "error",
        }
    }
}

pub fn send_callback(callback: &js_sys::Function, status: CallbackStatus, progress: f64) {
    let this = JsValue::null();
    let js_status = JsValue::from(status.as_str());
    let js_progress = JsValue::from(progress);
    let _ = callback.call2(&this, &js_status, &js_progress);
}
