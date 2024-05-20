use crate::util::js::log;
use js_sys::Uint8Array;
use serde::{Deserialize, Serialize};
use wasm_bindgen::prelude::*;
use wasm_bindgen_futures::JsFuture;
use web_sys::{Blob, Request, RequestInit, RequestMode, Response};

#[derive(Serialize, Deserialize)]
pub struct UploadUrlResponse {
    url: String,
}

pub async fn get_upload_url(api_url: &str, auth_token: &str, file_name: &str) -> Result<String, JsValue> {
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

pub async fn upload_file(data: &Vec<u8>, upload_url: String) -> Result<(), JsValue> {
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
