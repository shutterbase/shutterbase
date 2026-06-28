//! `image-wasm` — browser-side image processing for shutterbase: client thumbnailing,
//! presigned S3 upload, EXIF extraction, and the QR-based camera-clock time-sync.
//!
//! This module is the JS boundary; the heavy lifting lives in the sibling modules.
//! Exported function names and the serialized struct shapes are a contract with the
//! Vue frontend (`fileProcessor.ts`, `TimeOffsetCreate.vue`, `QrTimeCode.vue`) — keep
//! them stable.

mod callback;
mod error;
mod exif_meta;
mod filename;
mod imaging;
mod log;
mod qr;
mod time_offset;
mod upload;

use base64::{engine::general_purpose::STANDARD, Engine as _};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use uuid::Uuid;
use wasm_bindgen::prelude::*;

use crate::callback::{send, Status};
use crate::error::{Error, Result};
use crate::log::debug;
use crate::time_offset::TimeOffsetResult;

/// JPEG quality for generated thumbnails (the original full-res upload is sent
/// untouched). 85 is a good size/quality balance for downscaled previews.
const JPEG_QUALITY: u8 = 85;
/// QR generation canvas size, px.
const QR_SIZE: usize = 512;
/// Camera photos are downscaled to this longest-edge before QR detection — large
/// enough to keep the code legible, small enough to keep detection fast.
const QR_DECODE_MAX_EDGE: u32 = 1024;

/// Installs the panic hook so a Rust panic shows a readable message + stack in the
/// browser console instead of an opaque `RuntimeError: unreachable`.
#[wasm_bindgen(start)]
pub fn start() {
    console_error_panic_hook::set_once();
}

#[wasm_bindgen]
#[derive(Deserialize)]
pub struct FileProcessorOptions {
    file_name: String,
    dimensions: Vec<u32>,
    time_offsets: Vec<TimeOffsetResult>,
    copyright_tag: String,
    thumbnail_size: u32,
    api_url: String,
    // Binds the presigned upload URL request to an upload the caller may write to.
    upload_id: String,
}

#[wasm_bindgen]
#[derive(Serialize)]
pub struct FileProcessorResult {
    storage_id: String,
    thumbnail: String,
    original_size: u32,
    // Unix seconds as i64: a camera clock set before 1970 or far in the future
    // (the time-sync feature exists precisely because camera clocks are wrong)
    // would wrap/truncate under u32. JS numbers hold the full realistic range.
    camera_time_unix_seconds: i64,
    corrected_camera_time_unix_seconds: i64,
    computed_file_name: String,
    metadata: HashMap<String, String>,
    original_width: u32,
    original_height: u32,
}

/// Decode → read EXIF → compute corrected time + filename → thumbnail at each
/// dimension → upload original + thumbnails to S3. Reports progress via `callback`.
#[wasm_bindgen]
pub async fn process_file(
    file: js_sys::ArrayBuffer,
    options: JsValue,
    callback: &js_sys::Function,
) -> std::result::Result<JsValue, JsValue> {
    let started = js_sys::Date::now();
    send(callback, Status::Resizing, 0.0);

    let data = js_sys::Uint8Array::new(&file).to_vec();
    let options: FileProcessorOptions = serde_wasm_bindgen::from_value(options)
        .map_err(|e| Error::msg(format!("invalid processor options: {e}")))?;

    let source = imaging::decode(&data)?;
    let (original_width, original_height) = (source.width(), source.height());

    let metadata = exif_meta::read(&data)?;
    let camera_time = time_offset::camera_time(&metadata)?;
    let corrected_time = time_offset::corrected_camera_time(&metadata, &options.time_offsets)?;
    let computed_file_name =
        filename::calculate(&options.file_name, corrected_time, &options.copyright_tag)?;

    let object_id = Uuid::new_v4().to_string();
    let prefix = &object_id[..2];

    // Build the upload set: the untouched original, then a thumbnail per dimension
    // (resized from the full-res source for best quality). Storage keys must keep
    // the "<prefix>/<id>[-<dim>].jpg" shape the backend expects.
    // Move the original bytes into the upload set rather than cloning them — a
    // full-res copy is costly in the WASM heap. Capture the size first, since
    // `data` is consumed here.
    let original_size = data.len() as u32;
    let mut uploads: Vec<(String, Vec<u8>)> = Vec::with_capacity(options.dimensions.len() + 1);
    uploads.push((format!("{prefix}/{object_id}.jpg"), data));

    let mut dimensions = options.dimensions.clone();
    dimensions.sort_unstable();
    let steps = dimensions.len().max(1) as f64;
    let mut thumbnail = String::new();
    for (i, dimension) in dimensions.iter().rev().enumerate() {
        send(callback, Status::Resizing, (i as f64 + 1.0) / steps * 100.0);
        let resized = imaging::resize_within(&source, *dimension);
        let jpeg = imaging::encode_jpeg(&resized, JPEG_QUALITY)?;
        if *dimension == options.thumbnail_size {
            thumbnail = STANDARD.encode(&jpeg);
        }
        uploads.push((format!("{prefix}/{object_id}-{dimension}.jpg"), jpeg));
    }
    send(callback, Status::Resized, 100.0);

    let total_upload_size: usize = uploads.iter().map(|(_, bytes)| bytes.len()).sum();
    let mut uploaded = 0usize;
    for (key, bytes) in &uploads {
        let url = upload::get_upload_url(&options.api_url, key, &options.upload_id).await?;
        upload::upload_with_progress(bytes, total_upload_size, uploaded, url, callback).await?;
        uploaded += bytes.len();
    }
    send(callback, Status::Uploaded, 100.0);

    let result = FileProcessorResult {
        storage_id: object_id,
        thumbnail,
        original_size,
        camera_time_unix_seconds: camera_time.timestamp(),
        corrected_camera_time_unix_seconds: corrected_time.timestamp(),
        computed_file_name,
        metadata: metadata.tags,
        original_width,
        original_height,
    };

    debug(&format!("processed in {}ms", js_sys::Date::now() - started));
    Ok(serde_wasm_bindgen::to_value(&result)?)
}

/// Read EXIF metadata from an image. Returns the serialized [`exif_meta::ImageMetadata`].
#[wasm_bindgen]
pub async fn get_image_metadata(file: js_sys::ArrayBuffer) -> std::result::Result<JsValue, JsValue> {
    let data = js_sys::Uint8Array::new(&file).to_vec();
    let metadata = exif_meta::read(&data)?;
    Ok(serde_wasm_bindgen::to_value(&metadata)?)
}

/// Decode the QR code in an image and return its text payload.
#[wasm_bindgen]
pub async fn parse_qr_code(file: js_sys::ArrayBuffer) -> std::result::Result<JsValue, JsValue> {
    let data = js_sys::Uint8Array::new(&file).to_vec();
    let content = decode_qr(&data)?;
    Ok(serde_wasm_bindgen::to_value(&content)?)
}

/// Compute this camera's time offset from a photo of the server-time QR code.
#[wasm_bindgen]
pub async fn get_time_offset(file: js_sys::ArrayBuffer) -> std::result::Result<JsValue, JsValue> {
    let data = js_sys::Uint8Array::new(&file).to_vec();
    let content = decode_qr(&data)?;
    let metadata = exif_meta::read(&data)?;
    let offset = time_offset::from_qr(&metadata, &content)?;
    Ok(serde_wasm_bindgen::to_value(&offset)?)
}

#[derive(Serialize)]
struct QrCodeImageResult {
    time: u32,
    base64: String,
}

/// Render the server time as a PNG QR code (base64) for display to photographers.
#[wasm_bindgen]
pub async fn get_time_qr_code_image(time: u32) -> std::result::Result<JsValue, JsValue> {
    let png = qr::generate_png(&time.to_string(), QR_SIZE)?;
    let result = QrCodeImageResult {
        time,
        base64: STANDARD.encode(png),
    };
    Ok(serde_wasm_bindgen::to_value(&result)?)
}

/// Set the log threshold: `"debug" | "info" | "warn" | "error"`.
#[wasm_bindgen]
pub fn set_log_level(level: String) {
    log::set_level(&level);
}

/// Decode bytes → image → downscale → QR text.
fn decode_qr(data: &[u8]) -> Result<String> {
    let image = imaging::decode(data)?;
    let scaled = imaging::resize_within(&image, QR_DECODE_MAX_EDGE);
    qr::decode(&scaled)
}
