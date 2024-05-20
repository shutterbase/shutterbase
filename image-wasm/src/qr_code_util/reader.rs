use std::error::Error;

use crate::image::resizing::resize_image;
use bardecoder;
use image;

pub fn decode_qr_code(data: &Vec<u8>) -> Result<String, Box<dyn Error>> {
    let resized_image = match resize_image(data.clone(), 512) {
        Some(resized_image) => resized_image,
        None => return Err("Error resizing image".into()),
    };

    let image = match image::load_from_memory(&resized_image) {
        Ok(image) => image,
        Err(err) => return Err(err.to_string().into()),
    };

    let decoder = bardecoder::default_decoder();
    let results = decoder.decode(&image);

    if results.is_empty() {
        return Err("No QR code found".into());
    }

    if results.len() > 1 {
        return Err("Multiple QR codes found".into());
    }

    let result = match results.first() {
        Some(result) => match result {
            Ok(result) => result,
            Err(err) => return Err(err.to_string().into()),
        },
        None => return Err("No QR code found".into()),
    };

    Ok(result.to_string())
}
