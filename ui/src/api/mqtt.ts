import { http } from "src/boot/axios";

export interface MqttStatus {
  configured: boolean;
  reachable: boolean;
  error: string;
}

export interface MqttTestResult {
  ok: boolean;
  error?: string;
}

export async function getProjectMqttStatus(projectId: string): Promise<MqttStatus> {
  const { data } = await http.get<MqttStatus>(`/projects/${projectId}/mqtt/status`);
  return data;
}

export interface MqttTestPayload {
  broker: string;
  clientId?: string;
  username?: string;
  password?: string;
  topicPrefix?: string;
}

export async function testProjectMqtt(
  projectId: string,
  payload: MqttTestPayload,
): Promise<Record<string, MqttTestResult>> {
  const { data } = await http.post<Record<string, MqttTestResult>>(
    `/projects/${projectId}/mqtt/test`,
    payload,
  );
  return data;
}

export async function testProjectMqttWled(
  projectId: string,
  event: string,
): Promise<{ message: string; topic: string; payload: Record<string, unknown> }> {
  const { data } = await http.post(`/projects/${projectId}/mqtt/wled/test`, { event });
  return data;
}
