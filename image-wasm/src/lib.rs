mod api;
mod image;
mod qr_code_util;
mod util;

use crate::api::upload::{get_upload_url, upload_file};
use crate::image::metadata::read_image_metadata;
use crate::image::resizing::resize_image;
use crate::image::util::get_image_from_array_buffer;
use crate::qr_code_util::generator::get_time_qr_code;
use crate::qr_code_util::reader::decode_qr_code;

use crate::util::js::log;
use crate::util::time_offset::calculate_time_offset;

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

#[derive(Serialize, Deserialize)]
pub struct ProcessedFiles {
    id: String,
    thumbnail: String,
    original_size: u32,
    copyright: String,
    sizes: HashMap<u32, ProcessedFile>,
}

#[derive(Serialize, Deserialize)]
pub struct FileProcessorOptions {
    dimensions: Vec<u32>,
    thumbnail_size: u32,
    auth_token: String,
    api_url: String,
}

#[wasm_bindgen]
pub async fn process_file(file: js_sys::ArrayBuffer, js_options: JsValue) -> Result<JsValue, JsValue> {
    let overall_start_time = js_sys::Date::now();

    let data = js_sys::Uint8Array::new(&file).to_vec();

    let source_image = match get_image_from_array_buffer(file) {
        Ok(image) => image,
        Err(_) => {
            return Err(JsValue::UNDEFINED);
        }
    };

    let metadata = match read_image_metadata(&data) {
        Ok(metadata) => metadata,
        Err(_) => {
            return Err(JsValue::UNDEFINED);
        }
    };

    let original_size = source_image.as_bytes().len() as u32;

    let options: FileProcessorOptions = match serde_wasm_bindgen::from_value(js_options) {
        Ok(options) => options,
        Err(err) => {
            log("Error parsing options");
            log(&err.to_string());
            return Err(JsValue::UNDEFINED);
        }
    };

    let id = Uuid::new_v4();
    let mut processed_files = ProcessedFiles {
        id: id.to_string(),
        thumbnail: "".to_string(),
        original_size,
        copyright: metadata.copyright,
        sizes: HashMap::new(),
    };

    processed_files.sizes.insert(
        source_image.width(),
        ProcessedFile {
            dimension: source_image.width(),
            size: original_size,
            // data: data.clone(),
            // base64: general_purpose::STANDARD.encode(&data),
        },
    );

    let data = source_image.as_bytes().to_vec();

    let upload_url = get_upload_url(&options.api_url, &options.auth_token, &format!("{}.jpg", id)).await?;
    let upload_start_time = js_sys::Date::now();
    upload_file(&data, upload_url).await?;
    let upload_duration = js_sys::Date::now() - upload_start_time;
    log(&format!("Uploaded original in: {}ms", upload_duration));

    let mut dimensions: Vec<u32> = options.dimensions.clone();
    dimensions.sort();
    let inverted_dimensions: Vec<u32> = dimensions.iter().rev().cloned().collect();
    let mut last_converted_image_data = data.clone();

    log(&format!("Processing dimensions: {:?}", inverted_dimensions));

    for dimension in inverted_dimensions.iter() {
        // log(&format!("Processing dimension: {}", dimension));

        let upload_url = match get_upload_url(&options.api_url, &options.auth_token, &format!("{}-{}.jpg", id, dimension)).await {
            Ok(upload_url) => upload_url,
            Err(err) => {
                log("Error getting upload url");
                // log(err.as_string().unwrap().as_str());
                "error".to_string()
            }
        };

        log(&format!("Queried upload url: {}", upload_url));
        log(&format!("Converting to {}px", dimension));
        let start = js_sys::Date::now();
        let resized_image_data = match resize_image(last_converted_image_data, *dimension) {
            Some(data) => data,
            None => {
                log("Error resizing image");
                return Err(JsValue::UNDEFINED);
            }
        };

        last_converted_image_data = resized_image_data.clone();
        let duration = js_sys::Date::now() - start;
        log(&format!("Resized in: {}ms", duration));

        let resized_size = resized_image_data.len() as u32;
        let processed_file = ProcessedFile {
            dimension: *dimension,
            size: resized_size,
            // data: resized_image_data.clone(),
            // base64: general_purpose::STANDARD.encode(resized_image_data),
        };

        let upload_start_time = js_sys::Date::now();
        upload_file(&resized_image_data, upload_url).await?;
        let upload_duration = js_sys::Date::now() - upload_start_time;
        log(&format!("Uploaded {}px in: {}ms", dimension, upload_duration));

        processed_files.sizes.insert(*dimension, processed_file);

        if *dimension == options.thumbnail_size {
            processed_files.thumbnail = general_purpose::STANDARD.encode(resized_image_data)
        }
    }

    let result: JsValue = match serde_wasm_bindgen::to_value(&processed_files) {
        Ok(value) => value,
        Err(err) => {
            log("Error serializing file processing result");
            log(&err.to_string());
            JsValue::UNDEFINED
        }
    };

    let duration = js_sys::Date::now() - overall_start_time;
    log(&format!("Overall processing done in {}ms", duration));

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
            log("Error serializing file file metadata result");
            log(&err.to_string());
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
            log("Error serializing file file metadata result");
            log(&err.to_string());
            JsValue::UNDEFINED
        }
    };

    Ok(result)
}

#[wasm_bindgen]
pub async fn get_time_offset(file: js_sys::ArrayBuffer) -> Result<JsValue, JsValue> {
    let data = js_sys::Uint8Array::new(&file).to_vec();

    log(&format!("1/3 - Decoding QR code from image"));
    let qr_code_data = match decode_qr_code(&data) {
        Ok(qr_code_data) => qr_code_data,
        Err(err) => {
            log("Error decoding QR code image");
            log(&err.to_string());
            return Err(JsValue::UNDEFINED);
        }
    };

    log(&format!("2/3 - Reading exif metadata from image"));
    let metadata = match read_image_metadata(&data) {
        Ok(metadata) => metadata,
        Err(err) => {
            log("Error reading exif metadata from image");
            log(&err.to_string());
            return Err(JsValue::UNDEFINED);
        }
    };

    log(&format!("3/3 - Calculating time offset"));
    let time_offset = match calculate_time_offset(&metadata, &qr_code_data) {
        Ok(time_offset) => time_offset,
        Err(err) => {
            log("Error calculating time offset");
            log(&err.to_string());
            return Err(JsValue::UNDEFINED);
        }
    };

    let result: JsValue = match serde_wasm_bindgen::to_value(&time_offset) {
        Ok(value) => value,
        Err(err) => {
            log("Error serializing file file metadata result");
            log(&err.to_string());
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
            log("Error generating QR code image");
            log(&err.to_string());
            return Err(JsValue::UNDEFINED);
        }
    };

    let result: JsValue = match serde_wasm_bindgen::to_value(&qr_code_image_result) {
        Ok(value) => value,
        Err(err) => {
            log("Error serializing file processing result");
            log(&err.to_string());
            JsValue::UNDEFINED
        }
    };

    Ok(result)
}
