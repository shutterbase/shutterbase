use crate::util::callback_util::{send_callback, CallbackStatus};
use crate::util::logger::{debug, error, info, warn};
use js_sys::Uint8Array;
use serde::{Deserialize, Serialize};
use wasm_bindgen::prelude::*;
use wasm_bindgen_futures::JsFuture;
use web_sys::{Blob, Request, RequestInit, RequestMode, Response, XmlHttpRequest, XmlHttpRequestUpload};

#[derive(Serialize, Deserialize)]
pub struct UploadUrlResponse {
    url: String,
}

pub async fn get_upload_url(api_url: &str, auth_token: &str, file_name: &str) -> Result<String, JsValue> {
    debug(&format!("Getting upload url for {}", file_name));

    let mut opts = RequestInit::new();
    opts.method("GET");
    opts.mode(RequestMode::Cors);

    let url = format!("{}/upload-url?name={}", api_url, file_name);

    let request = Request::new_with_str_and_init(&url, &opts)?;
    request.headers().set("Authorization", auth_token)?;

    let window = web_sys::window().unwrap();
    let response: Response = match JsFuture::from(window.fetch_with_request(&request)).await {
        Ok(resp_value) => resp_value.dyn_into()?,
        Err(err) => {
            error("Error fetching upload url");
            error(err.as_string().unwrap().as_str());
            return Err(err);
        }
    };

    let json_data = JsFuture::from(response.json()?).await?;

    let upload_url_response = match serde_wasm_bindgen::from_value::<UploadUrlResponse>(json_data) {
        Ok(upload_url_response) => {
            debug(upload_url_response.url.as_str());
            upload_url_response
        }
        Err(err) => {
            error("Error parsing upload url response");
            error(&err.to_string());
            return Err(JsValue::UNDEFINED);
        }
    };

    Ok(upload_url_response.url)
}

pub async fn upload_file(data: &Vec<u8>, upload_url: String, file_name: String) -> Result<(), JsValue> {
    debug(&format!("Uploading file with {} bytes", data.len()));
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
    debug("Uploading file");
    let _response = JsFuture::from(window.fetch_with_request(&request)).await?;
    debug("File uploaded");

    Ok(())
}

pub async fn upload_file_with_progress(data: &Vec<u8>, total_upload_size: &usize, offset_size: &usize, upload_url: String, callback: &js_sys::Function) -> Result<(), JsValue> {
    debug(&format!("Uploading file with {} bytes", data.len()));

    let upload_fraction = data.len() as f64 / *total_upload_size as f64;
    let offset_progress = *offset_size as f64 / *total_upload_size as f64 * 100.0;

    // Convert Vec<u8> to Uint8Array for the Blob
    let uint8_array = Uint8Array::new_with_length(data.len() as u32);
    uint8_array.copy_from(&data);

    // Create a Blob with the file data
    let blob_parts = js_sys::Array::new();
    blob_parts.push(&uint8_array);
    let blob = Blob::new_with_u8_array_sequence(&blob_parts)?;

    // Create a new XMLHttpRequest
    let xhr = XmlHttpRequest::new()?;
    xhr.open_with_async("PUT", &upload_url, true)?;

    // Set up the onprogress event listener
    let upload: XmlHttpRequestUpload = xhr.upload().unwrap();
    let callback_clone = callback.clone();
    let onprogress_callback = Closure::wrap(Box::new(move |event: web_sys::ProgressEvent| {
        if event.length_computable() {
            let local_progress = (event.loaded() as f64 / event.total() as f64) * 100.0 * upload_fraction;
            let progress = offset_progress + local_progress;
            send_callback(&callback_clone, CallbackStatus::UPLOADING, progress);
        }
    }) as Box<dyn FnMut(_)>);
    upload.set_onprogress(Some(onprogress_callback.as_ref().unchecked_ref()));
    onprogress_callback.forget(); // Ensure the closure is kept alive

    // Set up the onload event listener to handle completion
    let callback_clone = callback.clone();
    let onload_callback = Closure::wrap(Box::new(move || {
        send_callback(&callback_clone, CallbackStatus::UPLOADED, 100.0);
    }) as Box<dyn FnMut()>);
    xhr.set_onload(Some(onload_callback.as_ref().unchecked_ref()));
    onload_callback.forget(); // Ensure the closure is kept alive

    // Set up the onerror event listener to handle errors
    let callback_clone = callback.clone();
    let onerror_callback = Closure::wrap(Box::new(move || {
        send_callback(&callback_clone, CallbackStatus::ERROR, 0.0);
    }) as Box<dyn FnMut()>);
    xhr.set_onerror(Some(onerror_callback.as_ref().unchecked_ref()));
    onerror_callback.forget(); // Ensure the closure is kept alive

    // Create a promise that resolves when the upload completes or errors
    let promise = js_sys::Promise::new(&mut |resolve, reject| {
        let onload = Closure::once_into_js(move || {
            resolve.call0(&JsValue::NULL).unwrap();
        });
        let onerror = Closure::once_into_js(move || {
            reject.call0(&JsValue::NULL).unwrap();
        });
        xhr.set_onload(Some(onload.as_ref().unchecked_ref()));
        xhr.set_onerror(Some(onerror.as_ref().unchecked_ref()));
    });

    // Send the request
    xhr.send_with_opt_blob(Some(&blob))?;

    // Await the promise to ensure the function waits for the upload to complete
    JsFuture::from(promise).await?;

    Ok(())
}
