import { writable } from 'svelte/store';
import api from './api';
export type Role = 'owner' | 'admin' | 'developer';
export type AuthUser = {
  id: string;
  username: string;
  email: string;
  role: Role;
  buckets: string[];
  createdAt: string;
};
export type AuthResponse = {
  enabled: boolean;
  authenticated: boolean;
  needsSetup: boolean;
  googleEnabled: boolean;
  user: AuthUser | null;
};
export const auth = writable<AuthResponse | null>(null);
export async function refreshAuth() {
  const status = await api.get<AuthResponse>('/auth/status');
  auth.set(status);
  return status;
}
