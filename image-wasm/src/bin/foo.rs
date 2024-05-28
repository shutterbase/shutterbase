use chrono::Local;

fn main() {
    eprintln!("offset: {:?}", Local::now().offset().to_string());    
}