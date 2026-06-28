//! Single error type for the crate. Internal functions return [`Result<T>`];
//! the `#[wasm_bindgen]` boundary converts it into a JavaScript `Error` via the
//! `From<Error> for JsValue` impl, so a thrown error surfaces in JS as
//! `Error { message: "<context>" }` instead of an opaque string.

use wasm_bindgen::JsValue;

#[derive(Debug, thiserror::Error)]
pub enum Error {
    #[error("io error: {0}")]
    Io(#[from] std::io::Error),

    #[error("image error: {0}")]
    Image(#[from] image::ImageError),

    #[error("exif error: {0}")]
    Exif(#[from] exif::Error),

    #[error("regex error: {0}")]
    Regex(#[from] regex::Error),

    #[error("qr generation failed: {0}")]
    QrGenerate(#[from] qrcode_generator::QRCodeError),

    #[error("{0}")]
    Message(String),
}

impl Error {
    /// Construct a free-text error.
    pub fn msg(message: impl Into<String>) -> Self {
        Error::Message(message.into())
    }
}

/// Surface a Rust error to JavaScript as a real `Error` object (with `.message`).
impl From<Error> for JsValue {
    fn from(error: Error) -> Self {
        js_sys::Error::new(&error.to_string()).into()
    }
}

pub type Result<T> = std::result::Result<T, Error>;
