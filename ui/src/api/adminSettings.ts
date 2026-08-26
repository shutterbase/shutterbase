import { http } from "src/boot/axios";

export interface MqttSettings {
  broker: string;
  clientId: string;
  username: string;
  password: string;
  topicPrefix: string;
}

export async function getMqttSettings(): Promise<MqttSettings> {
  const { data } = await http.get<MqttSettings>("/admin/settings/mqtt");
  return data;
}

export async function updateMqttSettings(settings: Partial<MqttSettings>): Promise<void> {
  await http.put("/admin/settings/mqtt", settings);
}
