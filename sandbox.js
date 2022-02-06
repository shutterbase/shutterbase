const ExifImage = require('exif').ExifImage;
const img = "0B8A9072.jpg"

async function getExifData(imagePath) {
  return new Promise((resolve, reject) => {
    try {
      new ExifImage({ image: imagePath }, function (error, exifData) {
        if (error)
          reject(error.message);
        else
          resolve(exifData);
      });
    } catch (error) {
      reject(error.message);
    }
  });
}

(async () => {
  const data = await getExifData(img)
  console.dir(data)
})()
