import { AxiosError, AxiosResponse } from "axios";

export interface ApiResponse {
  ok: boolean;
  status: number;
  code: string;
  message: string;
}

interface ApiError {
  code?: string;
  message: string;
}

const API_ERROR_CODES: { [key: string]: string } = {
  USER_ALREADY_EXISTS: "The user already exists",
};

function resolveErrorMessage(err: AxiosError<ApiError>): string {
  const code = err?.response?.data?.code || "";
  const message = err?.response?.data?.message;
  return API_ERROR_CODES[code] || message || err.toString();
}

export function resolveResponse(res: AxiosResponse): ApiResponse {
  return {
    ok: res.status >= 200 && res.status < 300,
    status: res.status,
    code: res.statusText,
    message: res.statusText,
  };
}

export function resolveErrorResponse(err: AxiosError<ApiError>): ApiResponse {
  return {
    ok: false,
    status: err?.response?.status || 0,
    code: err?.response?.data?.code || "",
    message: resolveErrorMessage(err),
  };
}
