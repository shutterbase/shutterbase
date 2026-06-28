//! S3 upload: fetch a presigned URL from the API, then PUT the bytes to it.
//!
//! Auth is the same-origin session cookie (`credentials: include`), so there is
//! no bearer token. Both the presign fetch and the PUT now check the HTTP status
//! — the old PUT resolved on any completed response, so a 403 from S3 looked like
//! a successful upload and produced an image record pointing at a missing object.

use crate::callback::{send, Status};
use crate::error::Error;
use crate::log::debug;
use js_sys::Uint8Array;
use serde::Deserialize;
use wasm_bindgen::prelude::*;
use wasm_bindgen_futures::JsFuture;
use web_sys::{
    Blob, Request, RequestCredentials, RequestInit, RequestMode, Response, XmlHttpRequest,
};

#[derive(Deserialize)]
struct UploadUrlResponse {
    url: String,
}

// api_url is "/api/v1", so this resolves to "/api/v1/upload-url?name=...&uploadId=...".
// uploadId binds the presign to an upload the caller may write to (server enforces).
fn upload_url_endpoint(api_url: &str, file_name: &str, upload_id: &str) -> String {
    format!("{api_url}/upload-url?name={file_name}&uploadId={upload_id}")
}

/// Ask the API for a presigned PUT URL for `file_name`, scoped to `upload_id`.
pub async fn get_upload_url(api_url: &str, file_name: &str, upload_id: &str) -> Result<String, JsValue> {
    let endpoint = upload_url_endpoint(api_url, file_name, upload_id);
    debug(&format!("requesting upload url for {file_name}"));

    let opts = RequestInit::new();
    opts.set_method("GET");
    opts.set_mode(RequestMode::Cors);
    opts.set_credentials(RequestCredentials::Include);

    let request = Request::new_with_str_and_init(&endpoint, &opts)?;
    let window = web_sys::window().ok_or_else(|| Error::msg("no window object"))?;
    let response: Response = JsFuture::from(window.fetch_with_request(&request))
        .await?
        .dyn_into()?;

    if !response.ok() {
        return Err(Error::msg(format!(
            "upload-url request failed: HTTP {}",
            response.status()
        ))
        .into());
    }

    let json = JsFuture::from(response.json()?).await?;
    let parsed: UploadUrlResponse = serde_wasm_bindgen::from_value(json)
        .map_err(|e| Error::msg(format!("invalid upload-url response: {e}")))?;
    Ok(parsed.url)
}

/// PUT `data` to the presigned URL, reporting progress through `callback`.
/// Resolves only on a 2xx response; a non-2xx status or network error rejects.
pub async fn upload_with_progress(
    data: &[u8],
    total_upload_size: usize,
    offset_size: usize,
    upload_url: String,
    callback: &js_sys::Function,
) -> Result<(), JsValue> {
    debug(&format!("uploading {} bytes", data.len()));

    let upload_fraction = data.len() as f64 / total_upload_size as f64;
    let offset_progress = offset_size as f64 / total_upload_size as f64 * 100.0;

    let array = Uint8Array::new_with_length(data.len() as u32);
    array.copy_from(data);
    let parts = js_sys::Array::new();
    parts.push(&array);
    let blob = Blob::new_with_u8_array_sequence(&parts)?;

    let xhr = XmlHttpRequest::new()?;
    xhr.open_with_async("PUT", &upload_url, true)?;

    // Progress ticks. forget() leaks one closure per upload — acceptable for a
    // short-lived request and matches the documented wasm-bindgen XHR pattern.
    let progress_cb = callback.clone();
    let on_progress = Closure::<dyn FnMut(web_sys::ProgressEvent)>::new(move |event: web_sys::ProgressEvent| {
        if event.length_computable() && event.total() > 0.0 {
            let local = event.loaded() / event.total() * 100.0 * upload_fraction;
            send(&progress_cb, Status::Uploading, offset_progress + local);
        }
    });
    xhr.upload()?
        .set_onprogress(Some(on_progress.as_ref().unchecked_ref()));
    on_progress.forget();

    // Completion: resolve on 2xx, reject otherwise. Registered exactly once.
    let xhr_for_promise = xhr.clone();
    let promise = js_sys::Promise::new(&mut |resolve, reject| {
        let xhr_on_load = xhr_for_promise.clone();
        let reject_on_load = reject.clone();
        let on_load = Closure::once_into_js(move || {
            let status = xhr_on_load.status().unwrap_or(0);
            if (200..300).contains(&status) {
                let _ = resolve.call0(&JsValue::NULL);
            } else {
                let error = js_sys::Error::new(&format!("upload failed: HTTP {status}"));
                let _ = reject_on_load.call1(&JsValue::NULL, &error);
            }
        });
        let on_error = Closure::once_into_js(move || {
            let error = js_sys::Error::new("upload network error");
            let _ = reject.call1(&JsValue::NULL, &error);
        });
        xhr_for_promise.set_onload(Some(on_load.as_ref().unchecked_ref()));
        xhr_for_promise.set_onerror(Some(on_error.as_ref().unchecked_ref()));
    });

    xhr.send_with_opt_blob(Some(&blob))?;
    JsFuture::from(promise).await?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn endpoint_targets_api_v1_upload_url() {
        assert_eq!(
            upload_url_endpoint("/api/v1", "ab/00000000-0000-0000-0000-000000000000.jpg", "up_abc123"),
            "/api/v1/upload-url?name=ab/00000000-0000-0000-0000-000000000000.jpg&uploadId=up_abc123"
        );
    }
}
