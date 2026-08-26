import { http } from "src/boot/axios";

export interface MqttStatus {
  connected: boolean;
}

export async function getStatus(): Promise<MqttStatus> {
  const { data } = await http.get<MqttStatus>("/mqtt/status");
  return data;
}
