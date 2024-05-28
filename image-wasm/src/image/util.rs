use crate::util::logger::{debug, error, info, warn};
use image::codecs::jpeg::JpegEncoder;
use image::ImageError;
use image::{io::Reader as ImageReader, DynamicImage};

use std::io::Cursor;

pub fn get_image_from_array_buffer(file: js_sys::ArrayBuffer) -> Result<DynamicImage, ImageError> {
    let data = js_sys::Uint8Array::new(&file).to_vec();

    let source_image = match get_dynamic_image_from_bytes(&data) {
        Ok(image) => image,
        Err(err) => {
            error("Error getting dynamic image from bytes");
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
            error("Error decoding image");
            error(&err.to_string());
            return Err(err);
        }
    }
}

pub fn get_jpeg_from_dynamic_image(image: DynamicImage) -> Result<Vec<u8>, ImageError> {
    let mut buffer = Vec::new();
    let mut jpeg_encoder = JpegEncoder::new_with_quality(&mut buffer, 100);
    jpeg_encoder.encode_image(&image)?;
    Ok(buffer)
}
