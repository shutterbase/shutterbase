//! EXIF extraction. Produces [`ImageMetadata`], whose `tags` map is the source
//! of truth for camera time (see [`crate::time_offset`]) and is surfaced to JS as
//! the image's `exifData`.

use crate::error::Result;
use exif::{Reader, Tag};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::io::{BufReader, Cursor};

#[derive(Serialize, Deserialize)]
pub struct ImageMetadata {
    pub filename: String,
    pub original_size: u32,
    pub copyright: String,
    pub created_at: String,
    pub date: String,
    /// Every EXIF field, keyed by tag name (e.g. `DateTimeOriginal`).
    pub tags: HashMap<String, String>,
}

pub fn read(data: &[u8]) -> Result<ImageMetadata> {
    let mut reader = BufReader::new(Cursor::new(data));
    let exif = Reader::new().read_from_container(&mut reader)?;

    let field = |tag: Tag| -> String {
        exif.fields()
            .find(|f| f.tag == tag)
            .map(|f| f.display_value().to_string())
            .unwrap_or_else(|| "Unknown".to_string())
    };

    let tags = exif
        .fields()
        .map(|f| {
            let key = f.tag.to_string().trim_matches('"').to_string();
            let value = f.display_value().to_string().trim_matches('"').to_string();
            (key, value)
        })
        .collect();

    Ok(ImageMetadata {
        filename: field(Tag::ImageDescription),
        original_size: data.len() as u32,
        copyright: field(Tag::Artist),
        created_at: field(Tag::DateTimeOriginal),
        date: field(Tag::DateTime),
        tags,
    })
}
