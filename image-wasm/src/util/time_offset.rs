use crate::image::metadata::ImageMetadata;
use crate::util::js::log;
use chrono::prelude::*;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize)]
pub struct TimeOffsetResult {
    pub time_offset: i64,
    pub server_time: i64,
    pub camera_time: i64,
}

pub fn calculate_time_offset(metadata: &ImageMetadata, qr_code_data: &String) -> Result<TimeOffsetResult, Box<dyn std::error::Error>> {
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

    let date_time_original_string = match metadata.tags.get("DateTimeOriginal") {
        Some(value) => value,
        None => "",
    };
    let date_time_original_time_zone_string = match metadata.tags.get("OffsetTimeOriginal") {
        Some(value) => value,
        None => "",
    };

    let camera_time_string = format!("{}{}", date_time_original_string, date_time_original_time_zone_string);
    log(&camera_time_string);

    let camera_time_fixed = match DateTime::parse_from_str(&camera_time_string, "%Y-%m-%d %H:%M:%S%z") {
        Ok(camera_time_fixed) => camera_time_fixed,
        Err(_) => {
            return Err("Error parsing camera time".into());
        }
    };

    let camera_time = camera_time_fixed.with_timezone(&Utc);

    let time_offset = (server_time - camera_time).num_seconds();

    let result = TimeOffsetResult {
        time_offset: time_offset,
        server_time: server_time.timestamp(),
        camera_time: camera_time.timestamp(),
    };

    Ok(result)
}
