// Maps any thrown value to the copy shown by UnexpectedErrorMessage. Pure so
// every failure shape stays unit-tested: axios/API errors, WASM js Errors
// (whose message/stack are non-enumerable and vanish in JSON.stringify),
// PocketBase-style field maps, plain strings.

type FieldError = { code?: string; message?: string };

export function errorHeadline(error: any): string {
  const data = error?.response?.data;
  if (data && typeof data === "object") {
    // Go API shape: { error: "conflict", message: "…" }
    if (typeof data.message === "string" && data.message !== "") {
      return data.message;
    }
    // Legacy PB shape: { field: { code, message } }
    const fields = Object.entries(data).filter(([, v]) => typeof (v as FieldError)?.message === "string");
    if (fields.length === 1) {
      return `Error on field '${fields[0][0]}': ${(fields[0][1] as FieldError).message}`;
    }
    if (fields.length > 1) {
      const messages = new Set(fields.map(([, v]) => (v as FieldError).message));
      return messages.size === 1 ? `Error on ${fields.length} fields: ${[...messages][0]}` : `Multiple errors on ${fields.length} fields`;
    }
  }
  // WASM and other native errors: the message is the whole story.
  if (typeof error?.message === "string" && error.message !== "") {
    return error.message;
  }
  if (typeof error === "string" && error !== "") {
    return error;
  }
  return "Unexpected Error";
}

export function errorDetails(error: any): string {
  if (error == null) {
    return "No details available";
  }
  if (error?.response) {
    return JSON.stringify({ message: error.message, status: error.response.status, data: error.response.data }, null, 2);
  }
  if (error instanceof Error) {
    return error.stack || `${error.name}: ${error.message}`;
  }
  return JSON.stringify(error, null, 2)?.trim() || String(error);
}
