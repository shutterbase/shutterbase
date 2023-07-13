import { api } from "src/boot/axios";
import { useLoginStore } from "src/stores/login-store";

export interface LoginRequest {
  email: string;
  password: string;
  rememberMe: boolean;
}

export interface RegistrationRequest {
  email: string;
  firstName: string;
  lastName: string;
  password: string;
}

export interface ConfirmationRequest {
  email: string;
  key: string;
}

export interface RequestPasswordResetRequest {
  email: String;
}

export interface PasswordResetRequest {
  email: String;
  key: String;
  password: String;
}

export interface RequestConfirmationEmailRequest {
  email: String;
}

export enum ResponseCode {
  OK = "ok",
  UNAUTHORIZED = "unauthorized",
  NETWORK_ERROR = "network_error",
  BAD_REQUEST = "bad_request",
  ERROR_PASSWORD = "error_hash_password",
  ERROR_SEND_EMAIL = "error_send_email",
  ERROR_CREATE_USER = "error_create_user",
  SERVER_ERROR = "server_error",
  USER_EXISTS = "user_exists",
  USER_NOT_FOUND = "user_not_found",
  EMAIL_REQUIRED = "email_required",
  PASSWORD_REQUIRED = "password_required",
  EMAIL_PASSWORD_REQUIRED = "email_password_required",
  LOGIN_PASSWORD_INVALID = "login_password_invalid",
  USER_NOT_ACTIVE = "user_not_active",
  EMAIL_NOT_VALIDATED = "email_not_validated",
  KEY_REQUIRED = "key_required",
  KEY_INVALID = "key_invalid",
  EMAIL_ALREADY_VALIDATED = "email_already_validated",
  TOO_MANY_REQUESTS = "too_many_requests",
  ERROR_RESET_PASSWORD = "error_reset_password",
  INVALID_TOKEN = "invalid_token",
  INVALID_REFRESH_TOKEN = "invalid_refresh_token",
  EXPIRED_TOKEN = "expired_token",
  MISSING_TOKEN = "missing_token",
}

export interface RegistrationResponse {
  code: ResponseCode;
}

export interface LoginResponse {
  code: ResponseCode;
}

function getErrorCode(error: any): ResponseCode {
  const errorCode = error?.response?.data?.error;
  const responseCode: ResponseCode = errorCode || ResponseCode.SERVER_ERROR;
  return responseCode;
}

export async function register(data: RegistrationRequest): Promise<RegistrationResponse> {
  console.log(`Registering user ${data.email}...`);
  try {
    await api.post("/register", data);
    return { code: ResponseCode.OK };
  } catch (error: any) {
    return { code: getErrorCode(error) };
  }
}

export async function confirmEmail(data: ConfirmationRequest) {
  // TODO: handle error codes from api
  try {
    const response = await api.post("/confirm", data);
    return { code: ResponseCode.OK };
  } catch (error) {
    return { code: getErrorCode(error) };
  }
}

function setTokenExpiration(data: { authTokenValidity: number; refreshTokenValidity: number }) {
  const { authTokenValidity, refreshTokenValidity } = data;
  const loginStore = useLoginStore();

  if (typeof authTokenValidity === "number") {
    const authTokenExpiration = new Date();
    authTokenExpiration.setSeconds(authTokenExpiration.getSeconds() + authTokenValidity);
    loginStore.authTokenExpiration = authTokenExpiration.getTime();
  }

  if (typeof refreshTokenValidity === "number") {
    const refreshTokenExpiration = new Date();
    refreshTokenExpiration.setSeconds(refreshTokenExpiration.getSeconds() + refreshTokenValidity);
    loginStore.refreshTokenExpiration = refreshTokenExpiration.getTime();
  }
}

export async function login(data: LoginRequest): Promise<LoginResponse> {
  try {
    const response = await api.post("/login", data);
    setTokenExpiration(response.data);
    await useLoginStore().setLoggedIn();
    return { code: ResponseCode.OK };
  } catch (error) {
    return { code: getErrorCode(error) };
  }
}

export async function logout() {
  useLoginStore().setLoggedOut();
  await api.post("/logout");
}

export async function requestPasswordReset(data: RequestPasswordResetRequest) {
  try {
    await api.post("/request-password-reset", data);
    return { code: ResponseCode.OK };
  } catch (error) {
    return { code: getErrorCode(error) };
  }
}

export async function passwordReset(data: PasswordResetRequest) {
  try {
    await api.post("/password-reset", data);
    return { code: ResponseCode.OK };
  } catch (error) {
    return { code: getErrorCode(error) };
  }
}

export async function refreshToken() {
  try {
    const response = await api.post("/refresh");
    setTokenExpiration(response.data);
    return { code: ResponseCode.OK };
  } catch (error) {
    const code = getErrorCode(error);
    console.log(`Error refreshing JWT: ${code}}`);
    // Both tokens might still be valid token might still be valid in case of network error
    // check for error code and only logout if error code is 401
    // logout is handled in the
    return { code };
  }
}

export async function requestConfirmationEmail(data: RequestConfirmationEmailRequest): Promise<{ code: ResponseCode }> {
  try {
    await api.post("/request-confirmation-email", data);
    return { code: ResponseCode.OK };
  } catch (error) {
    return { code: getErrorCode(error) };
  }
}
