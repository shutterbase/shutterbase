import { User } from "api/user";

export interface ApiKey {
  id: string;
  key: string;
  edges: {
    user: User;
  };
}
