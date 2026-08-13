//! RAW-file support for the time-offset QR path, via `rawler` — the library the
//! Lightroom-style editor PoC (branch `feature/editor`, crate `raw-wasm`) verified
//! on wasm32. Only the embedded-preview and metadata paths are used here: they run
//! on stock rawler, while a full develop would need the wasm timing patch carried
//! on that branch — and QR detection reads the camera's embedded JPEG just as well
//! as demosaiced pixels.

use crate::error::{Error, Result};
use crate::exif_meta::ImageMetadata;
use image::DynamicImage;
use rawler::decoders::RawDecodeParams;
use rawler::rawsource::RawSource;
use std::collections::HashMap;

/// Whether `rawler` recognises these bytes as a RAW container. JPEGs are sniffed
/// out by magic first: they are never RAW, but rawler's `get_decoder` claims some
/// maker JPEGs (Konica Minolta EXIF JPEGs route to its MRW decoder) and would
/// steal them from the working `image`-crate path.
pub fn is_raw(data: &[u8]) -> bool {
    if data.starts_with(&[0xFF, 0xD8, 0xFF]) {
        return false;
    }
    rawler::get_decoder(&RawSource::new_from_slice(data)).is_ok()
}

/// The camera's embedded full-size preview JPEG, decoded and ready for QR
/// detection.
pub fn preview(data: &[u8]) -> Result<DynamicImage> {
    let source = RawSource::new_from_slice(data);
    let decoder = rawler::get_decoder(&source)?;
    decoder
        .full_image(&source, &RawDecodeParams::default())?
        .ok_or_else(|| Error::msg("RAW file has no embedded preview image — photograph the QR code as JPEG instead"))
}

/// RAW maker metadata mapped onto the [`ImageMetadata`] shape the EXIF reader
/// produces, so `time_offset::camera_time` and the SPA read it unchanged.
pub fn metadata(data: &[u8]) -> Result<ImageMetadata> {
    let source = RawSource::new_from_slice(data);
    let decoder = rawler::get_decoder(&source)?;
    let md = decoder.raw_metadata(&source, &RawDecodeParams::default())?;

    let mut tags: HashMap<String, String> = HashMap::new();
    let mut insert = |key: &str, value: Option<String>| {
        if let Some(value) = value.filter(|v| !v.is_empty()) {
            tags.insert(key.to_string(), value);
        }
    };
    insert("DateTimeOriginal", md.exif.date_time_original.as_deref().map(exif_datetime_display));
    insert("OffsetTimeOriginal", md.exif.offset_time_original.clone());
    insert("SubSecTimeOriginal", md.exif.sub_sec_time_original.clone());
    insert("Make", Some(md.make.clone()));
    insert("Model", Some(md.model.clone()));
    insert("LensModel", md.exif.lens_model.clone().or_else(|| md.lens.as_ref().map(|l| l.lens_name.clone())));

    let created_at = tags.get("DateTimeOriginal").cloned().unwrap_or_else(|| "Unknown".to_string());
    Ok(ImageMetadata {
        filename: "Unknown".to_string(),
        original_size: data.len() as u32,
        copyright: "Unknown".to_string(),
        created_at: created_at.clone(),
        date: created_at,
        tags,
    })
}

/// EXIF datetimes come as `"YYYY:MM:DD HH:MM:SS"`; kamadak-exif *displays* them
/// with dashes and [`crate::time_offset::camera_time`] parses that display form —
/// match it. Only the date part's colons are separators to replace.
fn exif_datetime_display(value: &str) -> String {
    match value.split_once(' ') {
        Some((date, time)) => format!("{} {}", date.replace(':', "-"), time),
        None => value.to_string(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // Everything routes through `rawler::get_decoder`, which only knows RAW
    // containers — plain images must fall through to the standard `image` path.
    #[test]
    fn plain_images_are_not_raw() {
        let mut jpeg = Vec::new();
        image::codecs::jpeg::JpegEncoder::new_with_quality(&mut jpeg, 90)
            .encode(&[128u8; 16 * 16 * 3], 16, 16, image::ExtendedColorType::Rgb8)
            .expect("encode fixture");

        assert!(!is_raw(&jpeg), "rawler claimed a plain JPEG");
        assert!(preview(&jpeg).is_err());
        assert!(metadata(&jpeg).is_err());

        // the magic-byte guard, not rawler, must decide for JPEGs — rawler
        // accepts certain maker JPEGs (Konica Minolta) via its MRW path
        assert!(!is_raw(&[0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x00]));
    }

    #[test]
    fn exif_datetime_display_converts_only_the_date_colons() {
        assert_eq!(exif_datetime_display("2026:06:27 12:30:45"), "2026-06-27 12:30:45");
        // already-dashed or unexpected shapes pass through unharmed
        assert_eq!(exif_datetime_display("2026-06-27 12:30:45"), "2026-06-27 12:30:45");
        assert_eq!(exif_datetime_display("garbage"), "garbage");
    }
}
