import { nanoid } from "nanoid";
import { emitter } from "src/boot/mitt";
import { BACKEND_HOST, BACKEND_PORT, BACKEND_WEBSOCKET_PROTOCOL } from "src/boot/pocketbase";
import pb from "src/boot/pocketbase";

export type WebsocketMessage = {
  object: string;
  action: string;
  component: string;
  data: any;
};

let websocket: WebSocket;
export function isConnected() {
  if (!websocket) {
    return false;
  }
  return websocket.readyState === WebSocket.OPEN || websocket.readyState === WebSocket.CONNECTING;
}
const URL = `${BACKEND_WEBSOCKET_PROTOCOL}${BACKEND_HOST}${BACKEND_PORT}/api/ws`;

export function connect() {
  if (isConnected()) {
    return;
  }
  if (!pb.authStore.isValid) {
    setTimeout(connect, 1000);
  }
  websocket = new WebSocket(URL);
  websocket.onmessage = (event) => {
    const payload = JSON.parse(event.data);
    broadcast({
      object: payload.object,
      action: payload.action,
      component: payload.component,
      data: payload.data,
    });
  };
  websocket.onopen = () => {
    console.log("connected via websocket");
    emitter.emit("ws:open");
  };
  websocket.onerror = (error) => {
    console.log("websocket error: ", error);
    try {
      websocket.close();
    } catch (e) {
      console.log("error closing websocket: ", e);
    }
    setTimeout(connect, 1000);
  };
  websocket.onclose = () => {
    console.log("websocket got disconnected");
  };
}

export function disconnect() {
  websocket.close();
}

export type WebsocketMessageFilter = {
  object?: string;
  action?: string;
  component?: string;
};

type WebsocketListener = {
  id: string;
  filter: WebsocketMessageFilter;
  callback: (message: WebsocketMessage) => void;
};

const listeners: WebsocketListener[] = [];

export function on(filter: WebsocketMessageFilter, callback: (message: WebsocketMessage) => void): string {
  const id = nanoid();
  listeners.push({ id, filter, callback });
  return id;
}

export function off(id: string) {
  const index = listeners.findIndex((listener) => listener.id === id);
  if (index >= 0) {
    listeners.splice(index, 1);
  }
}

function broadcast(message: WebsocketMessage) {
  listeners.forEach((listener) => {
    if (
      (!listener.filter.object || listener.filter.object === message.object) &&
      (!listener.filter.action || listener.filter.action === message.action) &&
      (!listener.filter.component || listener.filter.component === message.component)
    ) {
      listener.callback(message);
    }
  });
}
