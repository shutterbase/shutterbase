import PocketBase from "pocketbase";

export const BACKEND_HOST = process.env.DEV ? "127.0.0.1" : window.location.hostname;
export const BACKEND_PORT = process.env.DEV ? ":8090" : ":443";
export const BACKEND_PROTOCOL = process.env.DEV ? "http://" : "https://";
export const BACKEND_WEBSOCKET_PROTOCOL = process.env.DEV ? "ws://" : "wss://";

const URL = `${BACKEND_PROTOCOL}${BACKEND_HOST}${BACKEND_PORT}`;
export default new PocketBase(URL);
