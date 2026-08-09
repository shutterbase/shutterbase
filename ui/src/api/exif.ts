import { http } from "src/boot/axios";
import type { ExifGroups } from "src/util/exifInspect";

// The file is read server-side in memory only (exiftool via stdin) — nothing is stored.
export async function inspect(file: File): Promise<ExifGroups> {
  const form = new FormData();
  form.append("file", file);
  const { data } = await http.post<{ metadata: ExifGroups }>("/exif/inspect", form);
  return data.metadata;
}
