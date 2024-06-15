export function loadFile(f: File): Promise<ArrayBuffer> {
  return new Promise((resolve, reject) => {
    const fileReader = new FileReader();
    fileReader.onload = (e: ProgressEvent<FileReader>) => {
      if (typeof e.target?.result === "object" && e.target?.result instanceof ArrayBuffer) {
        resolve(e.target?.result);
      } else {
        reject("Error loading file");
      }
    };
    fileReader.onerror = (e: ProgressEvent<FileReader>) => {
      reject(e);
    };
    fileReader.readAsArrayBuffer(f);
  });
}

export function fileSize(size: number): string {
  if (size < 1024) {
    return `${size} B`;
  } else if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  } else {
    return `${(size / 1024 / 1024).toFixed(1)} MB`;
  }
}
