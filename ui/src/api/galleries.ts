import { http } from "src/boot/axios";
import { Gallery, ListResponse } from "src/types/api";

export interface GalleryListParams {
  search?: string;
  limit?: number;
  offset?: number;
  sort?: string;
  order?: "asc" | "desc";
}

export type GalleryCreate = Pick<Gallery, "key" | "name"> & Partial<Omit<Gallery, "id" | "key" | "createdAt" | "updatedAt">>;
export type GalleryUpdate = Partial<Omit<Gallery, "id" | "key" | "createdAt" | "updatedAt">>;

export async function list(params: GalleryListParams = {}): Promise<ListResponse<Gallery>> {
  const { data } = await http.get<ListResponse<Gallery>>("/galleries", { params });
  return data;
}

export async function get(id: string): Promise<Gallery> {
  const { data } = await http.get<Gallery>(`/galleries/${id}`);
  return data;
}

export async function create(body: GalleryCreate): Promise<Gallery> {
  const { data } = await http.post<Gallery>("/galleries", body);
  return data;
}

export async function update(id: string, body: GalleryUpdate): Promise<Gallery> {
  const { data } = await http.put<Gallery>(`/galleries/${id}`, body);
  return data;
}

export async function remove(id: string): Promise<void> {
  await http.delete(`/galleries/${id}`);
}

// Mints a presigned PUT for one branding asset; returns the object key to store
// on the gallery (logoStorageId etc.) once the upload finished.
export async function createAssetUrl(id: string, contentType: string): Promise<{ uploadUrl: string; storageId: string }> {
  const { data } = await http.post<{ uploadUrl: string; storageId: string }>(`/galleries/${id}/assets`, { contentType });
  return data;
}

// Uploads a browser File straight to S3 and returns the stored key.
export async function uploadAsset(id: string, file: File): Promise<string> {
  const { uploadUrl, storageId } = await createAssetUrl(id, file.type);
  const res = await fetch(uploadUrl, { method: "PUT", body: file, headers: { "Content-Type": file.type } });
  if (!res.ok) throw new Error(`asset upload failed: ${res.status}`);
  return storageId;
}
