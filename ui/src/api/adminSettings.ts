import { http } from "src/boot/axios";

export interface MqttSettings {
  broker: string;
  clientId: string;
  username: string;
  password: string;
  topicPrefix: string;
  events: MqttEvents;
  presets: MqttPresets;
  triggerTags: string[];
}

export interface MqttEvents {
  uploadCreated: boolean;
  imageUploaded: boolean;
  ready: boolean;
  approved: boolean;
  rejected: boolean;
  imageRejected: boolean;
  tagAssigned: boolean;
}

export interface MqttPresets {
  uploadCreated: number;
  imageUploaded: number;
  ready: number;
  approved: number;
  rejected: number;
  imageRejected: number;
  tagAssigned: number;
}

export async function getProjectMqttSettings(projectId: string): Promise<MqttSettings> {
  const { data } = await http.get<MqttSettings>(`/projects/${projectId}/settings/mqtt`);
  return data;
}

export async function updateProjectMqttSettings(projectId: string, settings: Partial<MqttSettings>): Promise<void> {
  await http.put(`/projects/${projectId}/settings/mqtt`, settings);
}
