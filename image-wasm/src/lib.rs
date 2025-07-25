mod api;
mod image;
mod qr_code_util;
mod util;

use crate::api::upload::{get_upload_url, upload_file_with_progress};
use crate::image::metadata::read_image_metadata;
use crate::image::resizing::{get_image_size, resize_image};
use crate::image::util::get_image_from_array_buffer;
use crate::qr_code_util::generator::get_time_qr_code;
use crate::qr_code_util::reader::decode_qr_code;
use crate::util::logger::{debug, error, info};

use crate::util::callback_util::{send_callback, CallbackStatus};
use crate::util::time_offset::{calculate_qr_code_time_offset, get_camera_time};

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use uuid::Uuid;

use wasm_bindgen::prelude::*;

use base64::{engine::general_purpose, Engine as _};

#[derive(Serialize, Deserialize)]
pub struct ProcessedFile {
    dimension: u32,
    size: u32,
    // data: Vec<u8>,
    // base64: String,
}

#[wasm_bindgen]
#[derive(Serialize, Deserialize)]
pub struct FileProcessorResult {
    storage_id: String,
    thumbnail: String,
    original_size: u32,
    camera_time_unix_seconds: u32,
    corrected_camera_time_unix_seconds: u32,
    computed_file_name: String,
    metadata: HashMap<String, String>,
    original_width: u32,
    original_height: u32,
}

#[wasm_bindgen]
#[derive(Serialize, Deserialize)]
pub struct FileProcessorOptions {
    file_name: String,
    dimensions: Vec<u32>,
    time_offsets: Vec<util::time_offset::TimeOffsetResult>,
    copyright_tag: String,
    thumbnail_size: u32,
    auth_token: String,
    api_url: String,
}

#[wasm_bindgen]
pub async fn process_file(file: js_sys::ArrayBuffer, js_options: JsValue, callback: &js_sys::Function) -> Result<JsValue, JsValue> {
    let overall_start_time = js_sys::Date::now();
    send_callback(&callback, CallbackStatus::RESIZING, 0.0);

    let data = js_sys::Uint8Array::new(&file).to_vec();

    let options: FileProcessorOptions = match serde_wasm_bindgen::from_value(js_options) {
        Ok(options) => options,
        Err(err) => {
            error("Error parsing options");
            error(&err.to_string());
            return Err("Error parsing options".into());
        }
    };

    let source_image = match get_image_from_array_buffer(file) {
        Ok(image) => image,
        Err(_) => {
            return Err("Error reading image".into());
        }
    };

    let metadata = match read_image_metadata(&data) {
        Ok(metadata) => metadata,
        Err(_) => {
            return Err("Error reading image metadata".into());
        }
    };

    let original_size = source_image.as_bytes().len() as u32;

    let camera_time = match get_camera_time(&metadata) {
        Ok(camera_time) => camera_time,
        Err(err) => {
            return Err("Error getting camera time".into());
        }
    };
    let camera_time_unix_seconds = camera_time.timestamp() as u32;

    let corrected_camera_time = match util::time_offset::calculate_corrected_camera_time(&metadata, options.time_offsets) {
        Ok(corrected_camera_time) => corrected_camera_time,
        Err(_) => {
            return Err("Error calculating corrected camera time".into());
        }
    };
    let corrected_camera_time_unix_seconds = corrected_camera_time.timestamp() as u32;

    let computed_file_name = match util::filename::calculate_filename(options.file_name, corrected_camera_time, options.copyright_tag) {
        Ok(calculated_file_name) => calculated_file_name,
        Err(_) => {
            return Err("Error calculating file name".into());
        }
    };

    let object_id = Uuid::new_v4();
    let object_id_prefix = object_id.to_string()[..2].to_string();
    let (_source_image, source_width, source_height) = match get_image_size(data.clone()) {
        Some((image, width, height)) => (image, width, height),
        None => {
            error("Error getting image size");
            return Err("Error getting image size".into());
        }
    };
    let mut processed_files = FileProcessorResult {
        storage_id: object_id.to_string(),
        thumbnail: "".to_string(),
        original_size,
        camera_time_unix_seconds,
        corrected_camera_time_unix_seconds,
        computed_file_name,
        metadata: metadata.tags,
        original_width: source_width,
        original_height: source_height,
    };

    let mut upload_images: HashMap<String, Vec<u8>> = HashMap::new();
    upload_images.insert(format!("{}/{}.jpg", object_id_prefix.to_string(), object_id.to_string()), data.clone());
    let mut last_converted_image_data = data.clone();

    let mut dimensions: Vec<u32> = options.dimensions.clone();
    dimensions.sort();
    let inverted_dimensions: Vec<u32> = dimensions.iter().rev().cloned().collect();

    let mut progress = 0.0;
    let mut total_upload_size: usize = data.len();

    for dimension in inverted_dimensions.iter() {
        let start = js_sys::Date::now();

        progress += 100.0 / dimensions.len() as f64;
        send_callback(&callback, CallbackStatus::RESIZING, progress);
        debug(&format!("Converting to {}px", dimension));
        let resized_image_data = match resize_image(last_converted_image_data, *dimension) {
            Some(data) => data,
            None => {
                error("Error resizing image");
                return Err("Error resizing image".into());
            }
        };

        total_upload_size += &(resized_image_data.len());

        if *dimension == options.thumbnail_size {
            processed_files.thumbnail = general_purpose::STANDARD.encode(&resized_image_data)
        }

        last_converted_image_data = resized_image_data.clone();
        upload_images.insert(format!("{}/{}-{}.jpg", object_id_prefix, object_id.to_string(), dimension), resized_image_data);

        let duration = js_sys::Date::now() - start;
        debug(&format!("Resized in: {}ms", duration));
    }
    send_callback(&callback, CallbackStatus::RESIZED, 100.0);

    let mut upload_offset_size: usize = 0;

    for (filename, image_data) in upload_images.into_iter() {
        let upload_url = get_upload_url(&options.api_url, &options.auth_token, &filename).await?;

        debug(&format!("Queried upload url: {}", upload_url));

        let upload_start_time = js_sys::Date::now();
        match upload_file_with_progress(&image_data, &total_upload_size, &upload_offset_size, upload_url, callback).await {
            Ok(_) => {
                let upload_duration = js_sys::Date::now() - upload_start_time;
                debug(&format!("Uploaded {} in: {}ms", filename, upload_duration));
            }
            Err(err) => {
                error("Error uploading file");
                return Err(err);
            }
        }
        upload_offset_size += image_data.len();
    }
    send_callback(&callback, CallbackStatus::UPLOADED, 100.0);

    let result: JsValue = match serde_wasm_bindgen::to_value(&processed_files) {
        Ok(value) => value,
        Err(err) => {
            error("Error serializing file processing result");
            error(&err.to_string());
            return Err("Error serializing file processing result".into());
        }
    };

    let duration = js_sys::Date::now() - overall_start_time;
    debug(&format!("Overall processing done in {}ms", duration));

    Ok(result)
}

#[wasm_bindgen]
pub async fn get_image_metadata(file: js_sys::ArrayBuffer) -> Result<JsValue, JsValue> {
    let data = js_sys::Uint8Array::new(&file).to_vec();

    let metadata = match read_image_metadata(&data) {
        Ok(metadata) => metadata,
        Err(_) => {
            return Err(JsValue::UNDEFINED);
        }
    };

    let result: JsValue = match serde_wasm_bindgen::to_value(&metadata) {
        Ok(value) => value,
        Err(err) => {
            error("Error serializing file file metadata result");
            error(&err.to_string());
            JsValue::UNDEFINED
        }
    };

    Ok(result)
}

#[wasm_bindgen]
pub async fn parse_qr_code(file: js_sys::ArrayBuffer) -> Result<JsValue, JsValue> {
    let data = js_sys::Uint8Array::new(&file).to_vec();

    let qr_code_data = match decode_qr_code(&data) {
        Ok(qr_code_data) => qr_code_data,
        Err(_) => {
            return Err(JsValue::UNDEFINED);
        }
    };

    let result: JsValue = match serde_wasm_bindgen::to_value(&qr_code_data) {
        Ok(value) => value,
        Err(err) => {
            error("Error serializing file file metadata result");
            error(&err.to_string());
            JsValue::UNDEFINED
        }
    };

    Ok(result)
}

#[wasm_bindgen]
pub async fn get_time_offset(file: js_sys::ArrayBuffer) -> Result<JsValue, JsValue> {
    let data = js_sys::Uint8Array::new(&file).to_vec();

    info(&format!("1/3 - Decoding QR code from image"));
    let qr_code_data = match decode_qr_code(&data) {
        Ok(qr_code_data) => qr_code_data,
        Err(err) => {
            error("Error decoding QR code image");
            error(&err.to_string());
            return Err(JsValue::UNDEFINED);
        }
    };

    debug(&format!("2/3 - Reading exif metadata from image"));
    let metadata = match read_image_metadata(&data) {
        Ok(metadata) => metadata,
        Err(err) => {
            error("Error reading exif metadata from image");
            error(&err.to_string());
            return Err(JsValue::UNDEFINED);
        }
    };

    info(&format!("3/3 - Calculating time offset"));
    let time_offset = match calculate_qr_code_time_offset(&metadata, &qr_code_data) {
        Ok(time_offset) => time_offset,
        Err(err) => {
            error("Error calculating time offset");
            error(&err.to_string());
            return Err(JsValue::UNDEFINED);
        }
    };

    let result: JsValue = match serde_wasm_bindgen::to_value(&time_offset) {
        Ok(value) => value,
        Err(err) => {
            error("Error serializing file file metadata result");
            error(&err.to_string());
            JsValue::UNDEFINED
        }
    };

    Ok(result)
}

#[derive(Serialize, Deserialize)]
pub struct QrCodeImageResult {
    time: u32,
    base64: String,
}

#[wasm_bindgen]
pub async fn get_time_qr_code_image(time: u32) -> Result<JsValue, JsValue> {
    let qr_code_image_result = match get_time_qr_code(time) {
        Ok(png) => QrCodeImageResult {
            time,
            base64: general_purpose::STANDARD.encode(png),
        },
        Err(err) => {
            error("Error generating QR code image");
            error(&err.to_string());
            return Err(JsValue::UNDEFINED);
        }
    };

    let result: JsValue = match serde_wasm_bindgen::to_value(&qr_code_image_result) {
        Ok(value) => value,
        Err(err) => {
            error("Error serializing file processing result");
            error(&err.to_string());
            JsValue::UNDEFINED
        }
    };

    Ok(result)
}

#[wasm_bindgen]
pub async fn set_log_level(level: String) -> () {
    crate::util::logger::set_log_level_string(level);
}
