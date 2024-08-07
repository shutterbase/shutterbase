import PocketBase from "pocketbase";

export const BACKEND_HOST = process.env.DEV ? "127.0.0.1" : window.location.hostname;
export const BACKEND_PORT = process.env.DEV ? ":8090" : `:${window.location.port}`;
export const BACKEND_PROTOCOL = process.env.DEV ? "http://" : `${window.location.protocol}//`;
export const BACKEND_WEBSOCKET_PROTOCOL = process.env.DEV ? "ws://" : window.location.protocol === "http:" ? "ws://" : "wss://";

export const URL = `${BACKEND_PROTOCOL}${BACKEND_HOST}${BACKEND_PORT}`;
export const EXIF_WORKER_URL = process.env.DEV ? "http://localhost:8091" : URL;
const pb = new PocketBase(URL);
pb.autoCancellation(false);

export default pb;
