//! QR code generation (`qrcode_generator`) and decoding (`rqrr`).
//!
//! Decoding moved off the unmaintained `bardecoder` (which dragged an old `image`
//! version) onto `rqrr`, the maintained pure-Rust decoder. The generate→decode
//! roundtrip is covered by a native unit test so the swap is verified without a
//! browser; real camera-photo decoding (perspective, noise) still needs an
//! on-device check.

use crate::error::{Error, Result};
use image::DynamicImage;
use qrcode_generator::QrCodeEcc;

/// Render `content` as a PNG QR code, `size` px on a side.
pub fn generate_png(content: &str, size: usize) -> Result<Vec<u8>> {
    let png = qrcode_generator::to_png_to_vec(content, QrCodeEcc::Low, size)?;
    Ok(png)
}

/// Decode the single QR code in `image`. Errors if zero or more than one are found.
///
/// Uses `prepare_from_greyscale` (a closure over our luma buffer) rather than
/// rqrr's `image`-crate integration, so rqrr need not agree with us on an `image`
/// version — it pins 0.24, we use 0.25.
pub fn decode(image: &DynamicImage) -> Result<String> {
    let luma = image.to_luma8();
    let (width, height) = (luma.width() as usize, luma.height() as usize);
    let mut prepared = rqrr::PreparedImage::prepare_from_greyscale(width, height, |x, y| {
        luma.get_pixel(x as u32, y as u32)[0]
    });
    let grids = prepared.detect_grids();

    match grids.as_slice() {
        [] => Err(Error::msg("no QR code found")),
        [grid] => {
            let (_meta, content) = grid
                .decode()
                .map_err(|e| Error::msg(format!("QR decode failed: {e}")))?;
            Ok(content)
        }
        many => Err(Error::msg(format!(
            "expected exactly one QR code, found {}",
            many.len()
        ))),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn generate_then_decode_roundtrip() {
        let png = generate_png("1719500000", 512).expect("generate");
        let image = image::load_from_memory(&png).expect("load png");
        assert_eq!(decode(&image).expect("decode"), "1719500000");
    }

    #[test]
    fn decode_errors_when_no_qr_present() {
        let blank = DynamicImage::ImageLuma8(image::GrayImage::from_pixel(
            256,
            256,
            image::Luma([255]),
        ));
        assert!(decode(&blank).is_err());
    }
}
