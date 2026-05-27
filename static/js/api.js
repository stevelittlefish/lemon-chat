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
};

// Models
export const models = {
  list: () => request('GET', '/api/models'),
};

// Conversations
export const conversations = {
  list: () => request('GET', '/api/conversations'),
  create: (title, personaId) => request('POST', '/api/conversations', { title, persona_id: personaId }),
  delete: (id) => request('DELETE', `/api/conversations/${id}`),
};

// Messages
export const messages = {
  list: (conversationId) => request('GET', `/api/conversations/${conversationId}/messages`),
  // Returns an EventSource-like object. Calls onDelta(text) for each chunk, onDone() when finished.
  send: (conversationId, content, model, { onDelta, onDone, onError }) => {
    const url = `/api/conversations/${conversationId}/messages`;
    fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content, model }),
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
            const { delta, error } = JSON.parse(payload);
            if (error) { onError?.(new Error(error)); return; }
            if (delta) onDelta?.(delta);
          } catch { /* malformed chunk, skip */ }
        }
      }
      onDone?.();
    }).catch((err) => onError?.(err));
  },
};

// Personas
export const personas = {
  list: () => request('GET', '/api/personas'),
  create: (data) => request('POST', '/api/personas', data),
  update: (id, data) => request('PATCH', `/api/personas/${id}`, data),
  delete: (id) => request('DELETE', `/api/personas/${id}`),
};
