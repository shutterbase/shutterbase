use crate::image::metadata::ImageMetadata;
use crate::util::logger::{debug, error, info, warn};
use chrono::prelude::*;
use serde::{Deserialize, Serialize};
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
#[derive(Serialize, Deserialize)]
pub struct TimeOffsetResult {
    pub time_offset: i64,
    pub server_time: i64,
    pub camera_time: i64,
}

pub fn calculate_qr_code_time_offset(metadata: &ImageMetadata, qr_code_data: &String) -> Result<TimeOffsetResult, Box<dyn std::error::Error>> {
    let server_unix_time = match qr_code_data.parse::<i64>() {
        Ok(server_unix_time) => server_unix_time,
        Err(_) => {
            return Err("Error parsing server time".into());
        }
    };
    let server_time = match DateTime::from_timestamp(server_unix_time, 0) {
        Some(server_time) => server_time,
        None => {
            return Err("Error creating server time".into());
        }
    };

    let camera_time = match get_camera_time(metadata) {
        Ok(camera_time) => camera_time,
        Err(err) => {
            return Err(err);
        }
    };

    let time_offset = (server_time - camera_time).num_seconds();

    let result = TimeOffsetResult {
        time_offset: time_offset,
        server_time: server_time.timestamp(),
        camera_time: camera_time.timestamp(),
    };

    Ok(result)
}

pub fn calculate_corrected_camera_time(metadata: &ImageMetadata, time_offsets: Vec<TimeOffsetResult>) -> Result<DateTime<Utc>, Box<dyn std::error::Error>> {
    let camera_time = match get_camera_time(metadata) {
        Ok(camera_time) => camera_time,
        Err(err) => {
            return Err(err);
        }
    };

    let closest_time_offset = match find_closest_time_offset(camera_time, time_offsets) {
        Some(closest_time_offset) => closest_time_offset,
        None => {
            return Err("Error finding closest time offset".into());
        }
    };

    let corrected_camera_time = camera_time + chrono::Duration::seconds(closest_time_offset.time_offset);
    Ok(corrected_camera_time)
}

pub fn find_closest_time_offset(camera_time: DateTime<Utc>, time_offsets: Vec<TimeOffsetResult>) -> Option<TimeOffsetResult> {
    let mut closest_time_offset = Option::None;

    let mut closest_time_offset_difference = i64::MAX;

    for time_offset in time_offsets {
        let mut time_offset_camera_time = match DateTime::from_timestamp(time_offset.camera_time, 0) {
            Some(time_offset_camera_time) => time_offset_camera_time,
            None => {
                continue;
            }
        };
        let time_offset_difference = (camera_time - time_offset_camera_time).num_seconds().abs();
        if time_offset_difference < closest_time_offset_difference {
            closest_time_offset = Option::from(time_offset);
            closest_time_offset_difference = time_offset_difference;
        }
    }

    closest_time_offset
}

pub fn get_camera_time(metadata: &ImageMetadata) -> Result<DateTime<Utc>, Box<dyn std::error::Error>> {
    let date_time_original_string = match metadata.tags.get("DateTimeOriginal") {
        Some(value) => value,
        None => "",
    };

    let local_time_zone_offset = Local::now().offset().to_string();
    let date_time_original_time_zone_string = match metadata.tags.get("OffsetTimeOriginal") {
        Some(value) => value.as_str(),
        None => &local_time_zone_offset,
    };

    let camera_time_string = format!("{}{}", date_time_original_string, date_time_original_time_zone_string);
    debug(&camera_time_string);

    let camera_time_fixed = match DateTime::parse_from_str(&camera_time_string, "%Y-%m-%d %H:%M:%S%z") {
        Ok(camera_time_fixed) => camera_time_fixed,
        Err(_) => {
            return Err("Error parsing camera time".into());
        }
    };

    let camera_time = camera_time_fixed.with_timezone(&Utc);

    Ok(camera_time)
}
