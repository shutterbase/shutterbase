use qrcode_generator::{QRCodeError, QrCodeEcc};

pub fn get_time_qr_code(time: u32) -> Result<Vec<u8>, QRCodeError> {
    let result: Vec<u8> = qrcode_generator::to_png_to_vec(format!("{}", time), QrCodeEcc::Low, 512)?;
    Ok(result)
}
