import { http } from "src/boot/axios";

export interface MqttStatus {
  configured: boolean;
  connected: boolean;
}

export async function getProjectMqttStatus(projectId: string): Promise<MqttStatus> {
  const { data } = await http.get<MqttStatus>(`/projects/${projectId}/mqtt/status`);
  return data;
}
