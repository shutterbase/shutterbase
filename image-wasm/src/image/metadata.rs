use crate::util::logger::{debug, error, info, warn};

use exif;
use exif::Tag;
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, io::Cursor};

#[derive(Serialize, Deserialize)]
pub struct ImageMetadata {
    pub filename: String,
    pub original_size: u32,
    pub copyright: String,
    pub created_at: String,
    pub date: String,
    pub tags: HashMap<String, String>,
}

pub fn read_image_metadata(data: &Vec<u8>) -> Result<ImageMetadata, Box<dyn std::error::Error>> {
    let cursor = Cursor::new(&data);
    let mut buffer = std::io::BufReader::new(cursor);

    debug(format!("Data length: {}", data.len()).as_str());

    let exif = match exif::Reader::new().read_from_container(&mut buffer) {
        Ok(exif) => exif,
        Err(err) => {
            error("Error creating exif reader");
            error(&err.to_string());
            return Err(Box::new(err));
        }
    };

    let copyright = match exif.fields().find(|field| field.tag == Tag::Artist) {
        Some(field) => field.display_value().to_string(),
        None => "Unknown".to_string(),
    };
    let date = match exif.fields().find(|field| field.tag == Tag::DateTime) {
        Some(field) => field.display_value().to_string(),
        None => "Unknown".to_string(),
    };
    let filename = match exif.fields().find(|field| field.tag == Tag::ImageDescription) {
        Some(field) => field.display_value().to_string(),
        None => "Unknown".to_string(),
    };
    let created_at = match exif.fields().find(|field| field.tag == Tag::DateTimeOriginal) {
        Some(field) => field.display_value().to_string(),
        None => "Unknown".to_string(),
    };

    let mut tags: HashMap<String, String> = HashMap::new();

    exif.fields().for_each(|field| {
        let sanatized_tag = field.tag.to_string().trim_matches('"').to_string();
        let sanatized_value = field.display_value().to_string().trim_matches('"').to_string();
        tags.insert(sanatized_tag, sanatized_value);
    });

    let result = ImageMetadata {
        filename: filename.to_string(),
        original_size: data.len() as u32,
        copyright: copyright.to_string(),
        created_at: created_at.to_string(),
        date: date.to_string(),
        tags: tags,
    };

    Ok(result)
}
