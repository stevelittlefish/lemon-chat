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

// Auth
export const auth = {
  login: (username, password) => request('POST', '/api/auth/login', { username, password }),
  logout: () => request('POST', '/api/auth/logout'),
  me: () => request('GET', '/api/auth/me'),
  changePassword: (currentPassword, newPassword) =>
    request('PATCH', '/api/auth/password', { current_password: currentPassword, new_password: newPassword }),
};

// Models
export const models = {
  list: () => request('GET', '/api/models'),
};

// Conversations
export const conversations = {
  list: () => request('GET', '/api/conversations'),
  create: (title, model, characterId) => request('POST', '/api/conversations', { title, model, character_id: characterId }),
  delete: (id) => request('DELETE', `/api/conversations/${id}`),
};

// Messages
export const messages = {
  list: (conversationId) => request('GET', `/api/conversations/${conversationId}/messages`),
  // Returns an EventSource-like object. Calls onDelta(text) for each chunk, onDone() when finished.
  send: (conversationId, content, { onName, onDelta, onDone, onError }) => {
    const url = `/api/conversations/${conversationId}/messages`;
    fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content }),
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
            const { delta, error, name } = JSON.parse(payload);
            if (error) { onError?.(new Error(error)); return; }
            if (name) onName?.(name);
            if (delta) onDelta?.(delta);
          } catch { /* malformed chunk, skip */ }
        }
      }
      onDone?.();
    }).catch((err) => onError?.(err));
  },
};

// Characters
export const characters = {
  list:   ()         => request('GET',    '/api/characters'),
  create: (data)     => request('POST',   '/api/characters', data),
  update: (id, data) => request('PATCH',  `/api/characters/${id}`, data),
  delete: (id)       => request('DELETE', `/api/characters/${id}`),
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
