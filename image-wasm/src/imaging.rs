//! Image decode / resize / JPEG-encode, built on the `image` crate (0.25).
//!
//! The old code leaned on `fast_image_resize`, but without the `simd128` target
//! feature (not enabled by the default wasm build) it falls back to scalar code
//! — no faster than `image`'s own Lanczos3 resize, and one more dependency with
//! a churny API. So we use `image` directly.

use crate::error::Result;
use image::codecs::jpeg::JpegEncoder;
use image::imageops::FilterType;
use image::{DynamicImage, ImageReader};
use std::io::Cursor;

/// Decode an in-memory image (format auto-detected).
pub fn decode(bytes: &[u8]) -> Result<DynamicImage> {
    let image = ImageReader::new(Cursor::new(bytes))
        .with_guessed_format()?
        .decode()?;
    Ok(image)
}

/// Resize so the image fits within a `max_edge` × `max_edge` box, preserving the
/// aspect ratio. `DynamicImage::resize` never upscales past the box and keeps the
/// ratio, which is exactly the old hand-rolled `calculate_new_dimensions` logic.
pub fn resize_within(image: &DynamicImage, max_edge: u32) -> DynamicImage {
    image.resize(max_edge, max_edge, FilterType::Lanczos3)
}

/// Encode to JPEG at the given quality (0–100).
pub fn encode_jpeg(image: &DynamicImage, quality: u8) -> Result<Vec<u8>> {
    let mut buffer = Vec::new();
    let encoder = JpegEncoder::new_with_quality(&mut buffer, quality);
    image.write_with_encoder(encoder)?;
    Ok(buffer)
}

#[cfg(test)]
mod tests {
    use super::*;
    use image::{DynamicImage, RgbImage};

    fn sample(width: u32, height: u32) -> DynamicImage {
        DynamicImage::ImageRgb8(RgbImage::from_fn(width, height, |x, y| {
            image::Rgb([(x % 256) as u8, (y % 256) as u8, 128])
        }))
    }

    #[test]
    fn resize_preserves_aspect_ratio_within_box() {
        let resized = resize_within(&sample(2000, 1000), 512);
        // 2:1 image fit into 512x512 => 512x256.
        assert_eq!((resized.width(), resized.height()), (512, 256));
    }

    #[test]
    fn resize_fits_to_box_preserving_ratio() {
        // 5:4 fit into 512 => 512x410. resize() scales to fill the box (it will
        // upscale a small source), matching the original behaviour; in practice
        // camera files are always larger than the thumbnail sizes.
        let resized = resize_within(&sample(100, 80), 512);
        assert_eq!((resized.width(), resized.height()), (512, 410));
    }

    #[test]
    fn jpeg_roundtrip_decodes_back_to_same_dimensions() {
        let jpeg = encode_jpeg(&sample(640, 480), 90).expect("encode");
        let decoded = decode(&jpeg).expect("decode");
        assert_eq!((decoded.width(), decoded.height()), (640, 480));
    }
}
