//! Camera-clock reconciliation: the product's signature feature. A photographer's
//! camera clock drifts; a QR code of the server's unix time, photographed by the
//! camera, lets us compute that camera's offset and correct every capture.
//!
//! `TimeOffsetResult` is part of the JS contract (snake_case fields), consumed and
//! produced by `fileProcessor.ts` / `TimeOffsetCreate.vue`. The math here is a
//! straight port of the original — only the error handling was modernised.

use crate::error::{Error, Result};
use crate::exif_meta::ImageMetadata;
use chrono::prelude::*;
use serde::{Deserialize, Serialize};
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
#[derive(Serialize, Deserialize, Clone, Copy)]
pub struct TimeOffsetResult {
    pub time_offset: i64,
    pub server_time: i64,
    pub camera_time: i64,
}

/// Offset between this camera and the server, from a photo of the server-time QR.
pub fn from_qr(metadata: &ImageMetadata, qr_code_data: &str) -> Result<TimeOffsetResult> {
    let server_unix = qr_code_data
        .parse::<i64>()
        .map_err(|_| Error::msg("QR payload is not a unix timestamp"))?;
    let server_time = DateTime::from_timestamp(server_unix, 0)
        .ok_or_else(|| Error::msg("server timestamp out of range"))?;

    let camera_time = camera_time(metadata)?;

    Ok(TimeOffsetResult {
        time_offset: (server_time - camera_time).num_seconds(),
        server_time: server_time.timestamp(),
        camera_time: camera_time.timestamp(),
    })
}

/// Apply the closest-in-time offset to this image's camera time.
pub fn corrected_camera_time(
    metadata: &ImageMetadata,
    time_offsets: &[TimeOffsetResult],
) -> Result<DateTime<Utc>> {
    let camera_time = camera_time(metadata)?;
    let closest = closest_offset(camera_time, time_offsets)
        .ok_or_else(|| Error::msg("no time offset available"))?;
    Ok(camera_time + chrono::Duration::seconds(closest.time_offset))
}

/// The offset whose own camera-time is nearest to `camera_time`.
fn closest_offset(
    camera_time: DateTime<Utc>,
    time_offsets: &[TimeOffsetResult],
) -> Option<TimeOffsetResult> {
    time_offsets
        .iter()
        .filter_map(|offset| {
            let offset_time = DateTime::from_timestamp(offset.camera_time, 0)?;
            let distance = (camera_time - offset_time).num_seconds().abs();
            Some((distance, *offset))
        })
        .min_by_key(|(distance, _)| *distance)
        .map(|(_, offset)| offset)
}

/// Parse the camera's capture time from EXIF `DateTimeOriginal` (+ `OffsetTimeOriginal`,
/// falling back to the local zone when the camera didn't record one).
pub fn camera_time(metadata: &ImageMetadata) -> Result<DateTime<Utc>> {
    let date_time = metadata
        .tags
        .get("DateTimeOriginal")
        .map(String::as_str)
        .unwrap_or("");

    let local_offset = Local::now().offset().to_string();
    let zone = metadata
        .tags
        .get("OffsetTimeOriginal")
        .map(String::as_str)
        .unwrap_or(&local_offset);

    let combined = format!("{date_time}{zone}");
    let parsed = DateTime::parse_from_str(&combined, "%Y-%m-%d %H:%M:%S%z")
        .map_err(|_| Error::msg(format!("could not parse camera time from '{combined}'")))?;
    Ok(parsed.with_timezone(&Utc))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;

    fn metadata_with(date_time: &str, zone: &str) -> ImageMetadata {
        let mut tags = HashMap::new();
        tags.insert("DateTimeOriginal".to_string(), date_time.to_string());
        tags.insert("OffsetTimeOriginal".to_string(), zone.to_string());
        ImageMetadata {
            filename: "x".into(),
            original_size: 0,
            copyright: "x".into(),
            created_at: "x".into(),
            date: "x".into(),
            tags,
        }
    }

    #[test]
    fn camera_time_parses_explicit_zone() {
        let meta = metadata_with("2026-06-27 12:00:00", "+00:00");
        let time = camera_time(&meta).unwrap();
        assert_eq!(time.timestamp(), 1782561600);
    }

    #[test]
    fn from_qr_computes_offset_as_server_minus_camera() {
        let meta = metadata_with("2026-06-27 12:00:00", "+00:00");
        // server is 10s ahead of the camera
        let result = from_qr(&meta, "1782561610").unwrap();
        assert_eq!(result.time_offset, 10);
        assert_eq!(result.camera_time, 1782561600);
        assert_eq!(result.server_time, 1782561610);
    }

    #[test]
    fn corrected_time_applies_closest_offset() {
        let meta = metadata_with("2026-06-27 12:00:00", "+00:00");
        let offsets = [
            TimeOffsetResult { time_offset: 5, server_time: 0, camera_time: 1782561600 },
            TimeOffsetResult { time_offset: 999, server_time: 0, camera_time: 1 },
        ];
        let corrected = corrected_camera_time(&meta, &offsets).unwrap();
        assert_eq!(corrected.timestamp(), 1782561605);
    }

    #[test]
    fn from_qr_rejects_non_numeric_payload() {
        let meta = metadata_with("2026-06-27 12:00:00", "+00:00");
        assert!(from_qr(&meta, "not-a-number").is_err());
    }
}
