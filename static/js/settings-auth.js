import { auth } from './api.js';

export async function requireAuth() {
  try {
    return await auth.me();
  } catch {
    window.location.href = '/';
    return null;
  }
}
