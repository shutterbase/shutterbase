//! Computes the canonical upload filename: `<YYYYMMDD_HH-MM-SS>_<frame>_<copyright>`,
//! where `frame` is the last four-digit run in the camera's original filename.

use crate::error::{Error, Result};
use chrono::prelude::*;
use regex::Regex;

pub fn calculate(
    original_filename: &str,
    corrected_camera_time: DateTime<Utc>,
    copyright_tag: &str,
) -> Result<String> {
    let frame = last_four_digits(original_filename)?;
    // Format in UTC to match the backend's canonical filename
    // (service.computedFileName uses .UTC()). Using the browser-local zone here
    // made the upload-list preview disagree with the persisted name off-UTC.
    let timestamp = corrected_camera_time.format("%Y%m%d_%H-%M-%S");
    Ok(format!("{timestamp}_{frame}_{copyright_tag}"))
}

fn last_four_digits(input: &str) -> Result<String> {
    let re = Regex::new(r".*(\d{4}).*?")?;
    re.captures(input)
        .map(|cap| cap[1].to_string())
        .ok_or_else(|| Error::msg("no four consecutive digits found in filename"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn extracts_last_four_digit_run() {
        assert_eq!(last_four_digits("PS_04953.jpg").unwrap(), "4953");
        assert_eq!(last_four_digits("PS_04955.jpg").unwrap(), "4955");
        assert_eq!(last_four_digits("PS_04961 (2).jpg").unwrap(), "4961");
        assert_eq!(last_four_digits("PS_04955-EDIT.jpg").unwrap(), "4955");
    }

    #[test]
    fn errors_when_too_few_digits() {
        assert!(last_four_digits("PS_049.jpg").is_err());
    }

    #[test]
    fn builds_full_filename() {
        // UTC-formatted and deterministic regardless of the runner's timezone —
        // the timestamp portion is the contract with the backend filename.
        let time = DateTime::from_timestamp(0, 0).unwrap();
        let name = calculate("PS_04953.jpg", time, "JANE").unwrap();
        assert_eq!(name, "19700101_00-00-00_4953_JANE");
    }
}
