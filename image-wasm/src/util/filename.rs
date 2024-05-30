use chrono::prelude::*;
use regex::Regex;

pub fn calculate_filename(original_filename: String, corrected_camera_time: DateTime<Utc>, copyright_tag: String) -> Result<String, Box<dyn std::error::Error>> {
    let last_four_digits = extract_last_four_digits(&original_filename)?;
    let corrected_camera_time_string = corrected_camera_time.format("%Y%m%d_%H-%M-%S").to_string();
    let new_filename = format!("{}_{}_{}", corrected_camera_time_string, last_four_digits, copyright_tag);

    Ok(new_filename)
}

fn extract_last_four_digits(input: &str) -> Result<String, Box<dyn std::error::Error>> {
    let re = Regex::new(r"\d{4}").unwrap();
    let mut last_match = None;

    for mat in re.find_iter(input) {
        last_match = Some(mat.as_str());
    }

    match last_match {
        Some(digits) => Ok(digits.to_string()),
        None => Err("No four consecutive digits found in filename".into()),
    }
}
