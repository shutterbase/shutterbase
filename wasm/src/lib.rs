use exif::Tag;
// use photon_rs::transform::resize;
// use photon_rs::transform::SamplingFilter;
// use photon_rs::PhotonImage;

use fast_image_resize as fr;
use image::codecs::jpeg::JpegEncoder;
use image::ImageEncoder;
use image::{io::Reader as ImageReader, DynamicImage};

use std::io::{BufWriter, Cursor};
use std::num::NonZeroU32;

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use uuid::Uuid;

use js_sys::Uint8Array;
use wasm_bindgen::prelude::*;
use wasm_bindgen_futures::JsFuture;
use web_sys::{Blob, Request, RequestInit, RequestMode, Response};

use base64::{engine::general_purpose, Engine as _};

#[wasm_bindgen]
extern "C" {
    pub fn alert(s: &str);
}

#[wasm_bindgen]
extern "C" {
    #[wasm_bindgen(js_namespace = console)]
    fn log(s: &str);
}

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

#[derive(Serialize, Deserialize)]
pub struct UploadUrlResponse {
    url: String,
}

#[wasm_bindgen]
pub async fn process_file(file: js_sys::ArrayBuffer, js_options: JsValue) -> Result<JsValue, JsValue> {
    let overall_start_time = js_sys::Date::now();

    let data = js_sys::Uint8Array::new(&file).to_vec();

    let source_image = match get_dynamic_image_from_bytes(&data) {
        Ok(image) => image,
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

    let exif = match exif::Reader::new().read_from_container(&mut std::io::BufReader::new(Cursor::new(&data))) {
        Ok(exif) => exif,
        Err(err) => {
            log("Error creating exif reader");
            log(&err.to_string());
            return Err(JsValue::UNDEFINED);
        }
    };

    let copyright = match exif.fields().find(|field| field.tag == Tag::Artist) {
        Some(field) => field.display_value().to_string(),
        None => "Unknown".to_string(),
    };
    let date = match exif.fields().find(|field| field.tag == Tag::DateTime) {
        Some(field) => field.display_value().to_string(),
        None => "Unknown".to_string(),
    };

    let id = Uuid::new_v4();
    let mut processed_files = ProcessedFiles {
        id: id.to_string(),
        thumbnail: "".to_string(),
        original_size,
        copyright,
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
        let resized_image_data = resize_image(last_converted_image_data, *dimension);
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

fn resize_image(source_image_data: Vec<u8>, size: u32) -> Vec<u8> {
    let source_image = match get_dynamic_image_from_bytes(&source_image_data) {
        Ok(image) => image,
        Err(_) => {
            log("Error getting dynamic image from bytes");
            return vec![];
        }
    };

    let (width, height) = calculate_new_dimensions(source_image.width(), source_image.height(), size, size);
    log(&format!("Resizing to {}x{}", width, height));

    let src_width = NonZeroU32::new(source_image.width()).unwrap();
    let src_height = NonZeroU32::new(source_image.height()).unwrap();
    let src_image = fr::Image::from_vec_u8(src_width, src_height, source_image.to_rgb8().into_raw(), fr::PixelType::U8x3).unwrap();

    let dst_width = NonZeroU32::new(width).unwrap();
    let dst_height = NonZeroU32::new(height).unwrap();
    let mut dst_image = fr::Image::new(dst_width, dst_height, src_image.pixel_type());

    let mut dst_view = dst_image.view_mut();

    let mut resizer = fr::Resizer::new(fr::ResizeAlg::Convolution(fr::FilterType::Lanczos3));
    resizer.resize(&src_image.view(), &mut dst_view).unwrap();

    let mut result_buf = BufWriter::new(Vec::new());
    JpegEncoder::new(&mut result_buf)
        .write_image(dst_image.buffer(), dst_width.get(), dst_height.get(), image::ExtendedColorType::Rgb8)
        .unwrap();

    match result_buf.into_inner() {
        Ok(data) => data,
        Err(err) => {
            log("Error extracting resized image data");
            log(err.to_string().as_str());
            return vec![];
        }
    }
}

fn calculate_new_dimensions(width: u32, height: u32, max_width: u32, max_height: u32) -> (u32, u32) {
    let width_ratio = max_width as f32 / width as f32;
    let height_ratio = max_height as f32 / height as f32;
    let ratio = width_ratio.min(height_ratio);

    let new_width = (width as f32 * ratio).round() as u32;
    let new_height = (height as f32 * ratio).round() as u32;

    (new_width, new_height)
}

async fn get_upload_url(api_url: &str, auth_token: &str, file_name: &str) -> Result<String, JsValue> {
    log(&format!("Getting upload url for {}", file_name));

    let mut opts = RequestInit::new();
    opts.method("GET");
    opts.mode(RequestMode::Cors);

    let url = format!("{}/upload-url?name={}", api_url, file_name);
    // log(format!("Querying upload url: {}", url).as_str());
    // log(format!("Auth token: {}", auth_token).as_str());

    let request = Request::new_with_str_and_init(&url, &opts)?;
    request.headers().set("Authorization", auth_token)?;

    let window = web_sys::window().unwrap();
    let response: Response = match JsFuture::from(window.fetch_with_request(&request)).await {
        Ok(resp_value) => resp_value.dyn_into()?,
        Err(err) => {
            log("Error fetching upload url");
            log(err.as_string().unwrap().as_str());
            return Err(err);
        }
    };

    let json_data = JsFuture::from(response.json()?).await?;

    let upload_url_response = match serde_wasm_bindgen::from_value::<UploadUrlResponse>(json_data) {
        Ok(upload_url_response) => {
            log(upload_url_response.url.as_str());
            upload_url_response
        }
        Err(err) => {
            log("Error parsing upload url response");
            log(&err.to_string());
            return Err(JsValue::UNDEFINED);
        }
    };

    Ok(upload_url_response.url)
}

async fn upload_file(data: &Vec<u8>, upload_url: String) -> Result<(), JsValue> {
    // Convert Vec<u8> to Uint8Array for the Blob
    let uint8_array = Uint8Array::new_with_length(data.len() as u32);
    uint8_array.copy_from(&data);

    // Create a Blob with the file data
    let blob_parts = js_sys::Array::new();
    blob_parts.push(&uint8_array);
    let blob = Blob::new_with_u8_array_sequence(&blob_parts)?;

    let mut opts = RequestInit::new();
    opts.method("PUT");
    opts.body(Some(&blob));
    opts.mode(RequestMode::Cors);

    let request = Request::new_with_str_and_init(&upload_url, &opts)?;

    let window = web_sys::window().ok_or_else(|| JsValue::from_str("Could not obtain window"))?;
    let _response = JsFuture::from(window.fetch_with_request(&request)).await?;

    Ok(())
}

fn get_dynamic_image_from_bytes(data: &Vec<u8>) -> Result<DynamicImage, JsValue> {
    let mut image_reader = ImageReader::new(Cursor::new(data));
    image_reader.set_format(image::ImageFormat::Jpeg);
    match image_reader.decode() {
        Ok(image) => Ok(image),
        Err(err) => {
            log("Error decoding image");
            log(&err.to_string());
            return Err(JsValue::UNDEFINED);
        }
    }
}

#[wasm_bindgen]
pub async fn run(repo: String) -> Result<JsValue, JsValue> {
    let mut opts = RequestInit::new();
    opts.method("GET");
    opts.mode(RequestMode::Cors);

    let url = format!("https://api.github.com/repos/{}/branches/master", repo);

    let request = Request::new_with_str_and_init(&url, &opts)?;

    request.headers().set("Accept", "application/vnd.github.v3+json")?;

    let window = web_sys::window().unwrap();
    let resp_value = JsFuture::from(window.fetch_with_request(&request)).await?;

    // `resp_value` is a `Response` object.
    assert!(resp_value.is_instance_of::<Response>());
    let resp: Response = resp_value.dyn_into().unwrap();

    // Convert this other `Promise` into a rust `Future`.
    let json = JsFuture::from(resp.json()?).await?;

    // Send the JSON response back to JS.
    Ok(json)
}
