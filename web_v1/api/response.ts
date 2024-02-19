import { _AsyncData } from "nuxt/dist/app/composables/asyncData";

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

function resolveErrorMessage(err: any): string {
  const code = err?.response?.data?.code || "";
  const message = err?.response?.data?.message;
  return API_ERROR_CODES[code] || message || err.toString();
}

export function resolveResponse(res: any): ApiResponse {
  return {
    ok: res.status.value === "success",
    status: res.status.code,
    code: res.statusText,
    message: res.statusText,
  };
}

export function resolveErrorResponse(err: any): ApiResponse {
  return {
    ok: false,
    status: err?.response?.status || 0,
    code: err?.response?.data?.code || "",
    message: resolveErrorMessage(err),
  };
}
