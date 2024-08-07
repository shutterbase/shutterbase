use chrono::prelude::*;
use regex::Regex;

pub fn calculate_filename(original_filename: String, corrected_camera_time: DateTime<Utc>, copyright_tag: String) -> Result<String, Box<dyn std::error::Error>> {
    let last_four_digits = extract_last_four_digits(&original_filename)?;
    let corrected_camera_time_string = corrected_camera_time.with_timezone(&Local).format("%Y%m%d_%H-%M-%S").to_string();
    let new_filename = format!("{}_{}_{}", corrected_camera_time_string, last_four_digits, copyright_tag);

    Ok(new_filename)
}

fn extract_last_four_digits(input: &str) -> Result<String, Box<dyn std::error::Error>> {
    let re = Regex::new(r".*(\d{4}).*?")?;

    match re.captures(input) {
        Some(cap) => Ok(cap[1].to_string()),
        None => Err("No four consecutive digits found in filename".into()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_extract_last_four_digits_success() {
        let filename = "PS_04953.jpg";
        assert_eq!(extract_last_four_digits(filename).unwrap(), "4953");
    }

    #[test]
    fn test_extract_last_four_digits_too_few() {
        let filename = "PS_049.jpg";
        assert!(extract_last_four_digits(filename).is_err());
    }

    #[test]
    fn test_extract_last_four_digits_success_2() {
        let filename = "PS_04955.jpg";
        assert_eq!(extract_last_four_digits(filename).unwrap(), "4955");
    }

    #[test]
    fn test_extract_last_four_digits_success_file_duplicate() {
        let filename = "PS_04961 (2).jpg";
        assert_eq!(extract_last_four_digits(filename).unwrap(), "4961");
    }

    #[test]
    fn test_extract_last_four_digits_success_edit() {
        let filename = "PS_04955-EDIT.jpg";
        assert_eq!(extract_last_four_digits(filename).unwrap(), "4955");
    }
}
