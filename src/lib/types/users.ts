import type { Role } from '#lib/auth.ts';

export type User = {
  id: string;
  username: string;
  email: string;
  role: Role;
  buckets: string[];
  createdAt: string;
};
