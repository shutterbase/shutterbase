import { api } from "src/boot/axios";
import { resolveErrorResponse, resolveResponse, ApiResponse } from "./response";

export interface SingleResult<T> {
  item?: T;
  response: ApiResponse;
}

export interface ListResult<T> {
  items?: Array<T>;
  total?: number;
  response: ApiResponse;
}

export interface ListRequestOptions {
  limit?: number;
  offset?: number;
  search?: string;
  sort?: string;
  order?: string;
}

export async function requestSingle<T>(url: string): Promise<SingleResult<T>> {
  try {
    const response = await api.get(url);
    return {
      item: response.data as T,
      response: resolveResponse(response),
    };
  } catch (err: any) {
    console.log(err);
    return {
      response: resolveErrorResponse(err),
    };
  }
}

export async function requestList<T>(url: string, options: ListRequestOptions): Promise<ListResult<T>> {
  let params = [] as string[];
  if (options.limit) {
    params.push(`limit=${options.limit}`);
  }
  if (options.offset) {
    params.push(`offset=${options.offset}`);
  }
  if (options.search) {
    params.push(`search=${options.search}`);
  }
  if (options.sort) {
    params.push(`sort=${options.sort}`);
  }
  if (options.order) {
    params.push(`order=${options.order}`);
  }

  if (params.length > 0) {
    url += "?" + params.join("&");
  }
  try {
    const response = await api.get(url);
    return {
      items: response.data.items as Array<T>,
      total: response.data.total as number,
      response: resolveResponse(response),
    };
  } catch (err: any) {
    console.log(err);
    return {
      response: resolveErrorResponse(err),
    };
  }
}

export async function requestUpdate<T>(url: string, data: any): Promise<SingleResult<T>> {
  try {
    const response = await api.put(url, data);
    return {
      item: response.data as T,
      response: resolveResponse(response),
    };
  } catch (err: any) {
    console.log(err);
    return {
      response: resolveErrorResponse(err),
    };
  }
}
