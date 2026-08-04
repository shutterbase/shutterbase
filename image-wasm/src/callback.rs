//! Progress reporting back to the JS callback `(status: string, progress: number)`.
//! The status strings are part of the JS contract (see `fileProcessor.ts`).

use wasm_bindgen::JsValue;

#[derive(Clone, Copy)]
pub enum Status {
    Resizing,
    Resized,
    Uploading,
    Uploaded,
}

impl Status {
    fn as_str(self) -> &'static str {
        match self {
            Status::Resizing => "resizing",
            Status::Resized => "resized",
            Status::Uploading => "uploading",
            Status::Uploaded => "uploaded",
        }
    }
}

/// Invoke the JS progress callback. Errors from the JS side are ignored — a
/// failed progress tick must never abort processing.
pub fn send(callback: &js_sys::Function, status: Status, progress: f64) {
    let _ = callback.call2(
        &JsValue::NULL,
        &JsValue::from_str(status.as_str()),
        &JsValue::from_f64(progress),
    );
}
