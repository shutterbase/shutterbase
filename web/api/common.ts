import { resolveErrorResponse, resolveResponse, ApiResponse } from "./response";
import { DateTime } from "luxon";

export const API_BASE_URL = process.env.NODE_ENV === "development" ? "http://localhost:8081/api/v1" : "/api/v1";

export enum Method {
  GET = "GET",
  POST = "POST",
  PUT = "PUT",
  DELETE = "DELETE",
}

export function getFetchOptions(method: Method, body?: any, watch?: any[]) {
  const headers = useRequestHeaders(["cookie"]);
  const options: any = {
    method,
    headers,
    baseURL: API_BASE_URL,
    credentials: "include",
  };
  if (body) {
    options.body = body;
  }
  if (watch) {
    options.watch = watch;
  }
  return options;
}

export interface SingleResult<T> {
  item?: T;
  error: any;
  response: ApiResponse;
}

export interface ListResult<T> {
  items?: Array<T>;
  total?: number;
  response: ApiResponse;
}

interface ListResponse<T> {
  items: Array<T>;
  total: number;
}

export interface ListRequestOptions {
  limit?: number | Ref<number>;
  offset?: number | Ref<number>;
  search?: string | Ref<string>;
  sort?: string | Ref<string>;
  order?: string | Ref<string>;
  watch?: any[];
}

export async function requestSingle<T>(url: string): Promise<SingleResult<T>> {
  try {
    const response = await useFetch(url, getFetchOptions(Method.GET));
    return {
      item: response.data.value as T,
      error: response.error,
      response: resolveResponse(response),
    };
  } catch (err: any) {
    console.log(err);
    return {
      error: err,
      response: resolveErrorResponse(err),
    };
  }
}

export async function requestList<T>(url: string, options: ListRequestOptions): Promise<ListResult<T>> {
  let params = [] as string[];

  const applyOption = (key: string, value: string | number | Ref<string | number> | undefined) => {
    if (isRef(value)) {
      if (value.value) {
        params.push(`${key}=${value.value}`);
      }
    } else if (value) {
      params.push(`${key}=${value}`);
    }
  };

  applyOption("limit", options.limit);
  applyOption("offset", options.offset);
  applyOption("search", options.search);
  applyOption("sort", options.sort);
  applyOption("order", options.order);

  if (params.length > 0) {
    url += "?" + params.join("&");
  }
  try {
    const response = await useFetch(url, getFetchOptions(Method.GET, null, options.watch));

    const result = {
      response: resolveResponse(response),
      error: response.error,
    } as ListResult<T>;

    if (response.data.value) {
      const data = response.data.value as ListResponse<T>;
      result.items = data.items;
      result.total = data.total;
    }
    return result;
  } catch (err: any) {
    console.log(err);
    return {
      response: resolveErrorResponse(err),
    };
  }
}

export async function requestUpdate<T>(url: string, data: any): Promise<SingleResult<T>> {
  try {
    const response = await useFetch(url, getFetchOptions(Method.PUT, data));

    const result = {
      response: resolveResponse(response),
      error: response.error,
    } as SingleResult<T>;
    if (response.data.value) {
      result.item = response.data.value as T;
    }
    return result;
  } catch (err: any) {
    console.log(err);
    return {
      error: err,
      response: resolveErrorResponse(err),
    };
  }
}

export async function requestCreate<T>(url: string, data: any): Promise<SingleResult<T>> {
  try {
    const response = await useFetch(url, getFetchOptions(Method.POST, data));

    const result = {
      response: resolveResponse(response),
      error: response.error,
    } as SingleResult<T>;
    if (response.data.value) {
      result.item = response.data.value as T;
    }
    return result;
  } catch (err: any) {
    console.log(err);
    return {
      error: err,
      response: resolveErrorResponse(err),
    };
  }
}

export function getCreatedByString(item: any) {
  if (!item || !item.edges || !item.edges.createdBy) {
    return "unknown";
  } else {
    return `${item.edges.createdBy.firstName} ${item.edges.createdBy.lastName}`;
  }
}

export function getUpdatedByString(item: any) {
  if (!item || !item.edges || !item.edges.updatedBy) {
    return "unknown";
  } else {
    return `${item.edges.updatedBy.firstName} ${item.edges.updatedBy.lastName}`;
  }
}

export function getDateTimeString(dateTime: string): string {
  return DateTime.fromISO(dateTime).setLocale("en-GB").toLocaleString(DateTime.DATETIME_SHORT_WITH_SECONDS);
}

export function getDateString(dateTime: string): string {
  return DateTime.fromISO(dateTime).setLocale("en-GB").toLocaleString(DateTime.DATE_SHORT);
}

export function getTimeString(dateTime: string): string {
  return DateTime.fromISO(dateTime).setLocale("en-GB").toLocaleString(DateTime.TIME_WITH_SECONDS);
}
