use image::ImageError;
use image::{io::Reader as ImageReader, DynamicImage};

use std::io::Cursor;

use crate::util::js::log;

pub fn get_image_from_array_buffer(file: js_sys::ArrayBuffer) -> Result<DynamicImage, ImageError> {
    let data = js_sys::Uint8Array::new(&file).to_vec();

    let source_image = match get_dynamic_image_from_bytes(&data) {
        Ok(image) => image,
        Err(err) => {
            log("Error getting dynamic image from bytes");
            return Err(err);
        }
    };

    Ok(source_image)
}

pub fn get_dynamic_image_from_bytes(data: &Vec<u8>) -> Result<DynamicImage, ImageError> {
    let mut image_reader = ImageReader::new(Cursor::new(data));
    image_reader.set_format(image::ImageFormat::Jpeg);
    match image_reader.decode() {
        Ok(image) => Ok(image),
        Err(err) => {
            log("Error decoding image");
            log(&err.to_string());
            return Err(err);
        }
    }
}
