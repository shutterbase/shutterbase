use crate::util::logger::{debug, error, info, warn};
use fast_image_resize as fr;
use image::codecs::jpeg::JpegEncoder;
use image::ImageEncoder;

use std::io::BufWriter;
use std::num::NonZeroU32;

use super::util::get_dynamic_image_from_bytes;

pub fn calculate_new_dimensions(width: u32, height: u32, max_width: u32, max_height: u32) -> (u32, u32) {
    let width_ratio = max_width as f32 / width as f32;
    let height_ratio = max_height as f32 / height as f32;
    let ratio = width_ratio.min(height_ratio);

    let new_width = (width as f32 * ratio).round() as u32;
    let new_height = (height as f32 * ratio).round() as u32;

    (new_width, new_height)
}

pub fn get_image_size(image_data: Vec<u8>) -> Option<(image::DynamicImage, u32, u32)> {
    let image = match get_dynamic_image_from_bytes(&image_data) {
        Ok(image) => image,
        Err(_) => {
            error("Error getting dynamic image from bytes");
            return None;
        }
    };
    let width = image.width();
    let height = image.height();

    Some((image, width, height))
}

pub fn resize_image(source_image_data: Vec<u8>, size: u32) -> Option<Vec<u8>> {
    let (source_image, source_width, source_height) = match get_image_size(source_image_data) {
        Some((image, width, height)) => (image, width, height),
        None => {
            error("Error getting image size");
            return None;
        }
    };

    let (width, height) = calculate_new_dimensions(source_width, source_height, size, size);
    debug(&format!("Resizing to {}x{}", width, height));

    let src_width = match NonZeroU32::new(source_image.width()) {
        Some(width) => width,
        None => {
            error("Error getting source image width");
            return None;
        }
    };
    let src_height = match NonZeroU32::new(source_image.height()) {
        Some(height) => height,
        None => {
            error("Error getting source image height");
            return None;
        }
    };
    let src_image = match fr::Image::from_vec_u8(src_width, src_height, source_image.to_rgb8().into_raw(), fr::PixelType::U8x3) {
        Ok(image) => image,
        Err(err) => {
            error("Error creating source image");
            error(err.to_string().as_str());
            return None;
        }
    };

    let dst_width = match NonZeroU32::new(width) {
        Some(width) => width,
        None => {
            error("Error getting destination image width");
            return None;
        }
    };
    let dst_height = match NonZeroU32::new(height) {
        Some(height) => height,
        None => {
            error("Error getting destination image height");
            return None;
        }
    };
    let mut dst_image = fr::Image::new(dst_width, dst_height, src_image.pixel_type());

    let mut dst_view = dst_image.view_mut();

    let mut resizer = fr::Resizer::new(fr::ResizeAlg::Convolution(fr::FilterType::Lanczos3));
    match resizer.resize(&src_image.view(), &mut dst_view) {
        Ok(_) => (),
        Err(err) => {
            error("Error resizing image");
            error(err.to_string().as_str());
            return None;
        }
    }

    let mut result_buf = BufWriter::new(Vec::new());
    match JpegEncoder::new(&mut result_buf).write_image(dst_image.buffer(), dst_width.get(), dst_height.get(), image::ColorType::Rgb8) {
        Ok(_) => (),
        Err(err) => {
            error("Error encoding resized image");
            error(err.to_string().as_str());
            return None;
        }
    };

    match result_buf.into_inner() {
        Ok(data) => Some(data),
        Err(err) => {
            error("Error extracting resized image data");
            error(err.to_string().as_str());
            return None;
        }
    }
}
