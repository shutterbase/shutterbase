use fast_image_resize as fr;
use image::codecs::jpeg::JpegEncoder;
use image::ImageEncoder;

use std::io::BufWriter;
use std::num::NonZeroU32;
use std::vec;

use crate::util::js::log;

use super::util::get_dynamic_image_from_bytes;

pub fn calculate_new_dimensions(width: u32, height: u32, max_width: u32, max_height: u32) -> (u32, u32) {
    let width_ratio = max_width as f32 / width as f32;
    let height_ratio = max_height as f32 / height as f32;
    let ratio = width_ratio.min(height_ratio);

    let new_width = (width as f32 * ratio).round() as u32;
    let new_height = (height as f32 * ratio).round() as u32;

    (new_width, new_height)
}

pub fn resize_image(source_image_data: Vec<u8>, size: u32) -> Vec<u8> {
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
        .write_image(dst_image.buffer(), dst_width.get(), dst_height.get(), image::ColorType::Rgb8)
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
