import { http } from "src/boot/axios";

export interface MqttSettings {
  broker: string;
  clientId: string;
  username: string;
  password: string;
  topicPrefix: string;
}

export async function getProjectMqttSettings(projectId: string): Promise<MqttSettings> {
  const { data } = await http.get<MqttSettings>(`/projects/${projectId}/settings/mqtt`);
  return data;
}

export async function updateProjectMqttSettings(projectId: string, settings: Partial<MqttSettings>): Promise<void> {
  await http.put(`/projects/${projectId}/settings/mqtt`, settings);
}
