// Fetch wrappers for the lemon-chat REST API.

async function request(method, path, body) {
  const opts = { method, headers: {} };
  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  const res = await fetch(path, opts);
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data;
}

async function uploadFile(method, path, file) {
  const form = new FormData();
  form.append('avatar', file);
  const res = await fetch(path, { method, body: form });
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data;
}

// Auth
export const auth = {
  login: (username, password) => request('POST', '/api/auth/login', { username, password }),
  logout: () => request('POST', '/api/auth/logout'),
  me: () => request('GET', '/api/auth/me'),
  changePassword: (currentPassword, newPassword) =>
    request('PATCH', '/api/auth/password', { current_password: currentPassword, new_password: newPassword }),
  updateProfile: (displayName) =>
    request('PATCH', '/api/auth/profile', { display_name: displayName }),
  uploadAvatar: (file) => uploadFile('PUT', '/api/auth/avatar', file),
  deleteAvatar: () => request('DELETE', '/api/auth/avatar'),
};

// Models
export const models = {
  list: () => request('GET', '/api/models'),
};

// Conversations
export const conversations = {
  list: () => request('GET', '/api/conversations'),
  create: (title, model, characterId) => request('POST', '/api/conversations', { title, model, character_id: characterId }),
  fork: (id, messageId) => request('POST', `/api/conversations/${id}/fork`, { message_id: messageId }),
  update: (id, data) => request('PATCH', `/api/conversations/${id}`, data),
  delete: (id) => request('DELETE', `/api/conversations/${id}`),
  regenerateTitle: (id) => request('POST', `/api/conversations/${id}/regenerate-title`),
};

// Messages
export const messages = {
  list: (conversationId) => request('GET', `/api/conversations/${conversationId}/messages`),
  firstMessage: (conversationId, characterId = null) => {
    const body = characterId != null ? { character_id: characterId } : {};
    return request('POST', `/api/conversations/${conversationId}/first-message`, body);
  },
  // selection: { type: 'model', name } | { type: 'character', id } | null
  send: (conversationId, content, selection, { onName, onDelta, onDone, onError, onStats, onMessageId }) => {
    const url = `/api/conversations/${conversationId}/messages`;
    const body = { content };
    if (selection?.type === 'model') body.model = selection.name;
    if (selection?.type === 'character') body.character_id = selection.id;
    fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }).then(async (res) => {
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        onError?.(new Error(data.error || res.statusText));
        return;
      }
      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop(); // keep incomplete line
        for (const line of lines) {
          if (!line.startsWith('data: ')) continue;
          const payload = line.slice(6);
          if (payload === '[DONE]') { onDone?.(); return; }
          try {
            const { delta, error, name, stats, message_id } = JSON.parse(payload);
            if (error) { onError?.(new Error(error)); return; }
            if (name) onName?.(name);
            if (delta) onDelta?.(delta);
            if (stats) onStats?.(stats);
            if (message_id) onMessageId?.(message_id);
          } catch { /* malformed chunk, skip */ }
        }
      }
      onDone?.();
    }).catch((err) => onError?.(err));
  },
};

// Characters
export const characters = {
  list:         ()         => request('GET',    '/api/characters'),
  get:          (id)       => request('GET',    `/api/characters/${id}`),
  create:       (data)     => request('POST',   '/api/characters', data),
  update:       (id, data) => request('PATCH',  `/api/characters/${id}`, data),
  delete:       (id)       => request('DELETE', `/api/characters/${id}`),
  uploadAvatar: (id, file) => uploadFile('PUT', `/api/characters/${id}/avatar`, file),
  deleteAvatar: (id)       => request('DELETE', `/api/characters/${id}/avatar`),
};

// Admin
export const admin = {
  users: {
    list: () => request('GET', '/api/admin/users'),
    create: (data) => request('POST', '/api/admin/users', data),
    update: (id, data) => request('PATCH', `/api/admin/users/${id}`, data),
    delete: (id) => request('DELETE', `/api/admin/users/${id}`),
  },
};
