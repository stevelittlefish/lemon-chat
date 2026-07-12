// Fetch wrappers for the lemon-chat REST API.

// Reads an SSE response body, calling onEvent(parsed) for each JSON chunk.
// onEvent may return false to stop early (without calling onDone).
// Calls onDone when [DONE] is received or the stream ends naturally.
async function consumeSSE(res, onEvent, onDone) {
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop();
    for (const line of lines) {
      if (!line.startsWith('data: ')) continue;
      const payload = line.slice(6);
      if (payload === '[DONE]') { onDone?.(); return; }
      try {
        if (onEvent(JSON.parse(payload)) === false) return;
      } catch { /* malformed chunk, skip */ }
    }
  }
  onDone?.();
}

async function request(method, path, body) {
  const opts = { method, headers: {} };
  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  if (method !== 'GET' && method !== 'HEAD') {
    opts.headers['X-Requested-With'] = 'XMLHttpRequest';
  }
  const res = await fetch(path, opts);
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) {
    const err = new Error(data.error || res.statusText);
    err.status = res.status;
    throw err;
  }
  return data;
}

async function uploadFile(method, path, file) {
  const form = new FormData();
  form.append('avatar', file);
  const res = await fetch(path, { method, body: form, headers: { 'X-Requested-With': 'XMLHttpRequest' } });
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
  list: (mode) => request('GET', mode ? `/api/models?mode=${encodeURIComponent(mode)}` : '/api/models'),
};

// Tools
export const tools = {
  list: () => request('GET', '/api/tools'),
};

// Notes
export const notes = {
  list:        (prefix)           => request('GET',    prefix ? `/api/notes?prefix=${encodeURIComponent(prefix)}` : '/api/notes'),
  get:         (id)               => request('GET',    `/api/notes/${id}`),
  upsert:      (key, value, readOnly) => request('PUT', '/api/notes', { key, value, ...(readOnly !== undefined ? { read_only: readOnly } : {}) }),
  delete:      (id)               => request('DELETE', `/api/notes/${id}`),
  setReadOnly: (id, readOnly)     => request('PATCH',  `/api/notes/${id}/read-only`, { read_only: readOnly }),
};

// Conversations
export const conversations = {
  list: (offset = 0) => request('GET', `/api/conversations?limit=30&offset=${offset}`),
  create: (title, model, characterId) => request('POST', '/api/conversations', { title, model, character_id: characterId }),
  fork: (id, messageId) => request('POST', `/api/conversations/${id}/fork`, { message_id: messageId }),
  importChat: (data) => request('POST', '/api/conversations/import_chat', data),
  update: (id, data) => request('PATCH', `/api/conversations/${id}`, data),
  delete: (id) => request('DELETE', `/api/conversations/${id}`),
  regenerateTitle: (id) => request('POST', `/api/conversations/${id}/regenerate-title`),
};

// Messages
export const messages = {
  list: (conversationId) => request('GET', `/api/conversations/${conversationId}/messages`),
  context: (conversationId, messageId) => request('GET', `/api/conversations/${conversationId}/messages/${messageId}/context`),
  firstMessage: (conversationId, characterId = null) => {
    const body = characterId != null ? { character_id: characterId } : {};
    return request('POST', `/api/conversations/${conversationId}/first-message`, body);
  },
  // selection: { type: 'model', name } | { type: 'character', id } | null
  // Returns an abort() function that cancels the in-flight request.
  send: (conversationId, content, selection, { onName, onDelta, onDone, onAborted, onError, onStats, onMessageId, onToolCall, onToolResult, onAttachment, onNewTurn }) => {
    const url = `/api/conversations/${conversationId}/messages`;
    const body = { content };
    if (selection?.type === 'model') body.model = selection.name;
    if (selection?.type === 'character') body.character_id = selection.id;
    const ctrl = new AbortController();
    fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
      body: JSON.stringify(body),
      signal: ctrl.signal,
    }).then(async (res) => {
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        onError?.(new Error(data.error || res.statusText));
        return;
      }
      await consumeSSE(res, (ev) => {
        const { delta, error, name, stats, message_id, tool_call, tool_result, attachment, new_turn } = ev;
        if (error) { onError?.(new Error(error)); return false; }
        if (name) onName?.(name);
        if (delta) onDelta?.(delta);
        if (stats) onStats?.(stats);
        if (message_id) onMessageId?.(message_id);
        if (tool_call) onToolCall?.(tool_call);
        if (tool_result) onToolResult?.(tool_result);
        if (attachment) onAttachment?.(attachment);
        if (new_turn) onNewTurn?.(new_turn);
      }, onDone);
    }).catch((err) => {
      if (err.name === 'AbortError') onAborted?.();
      else onError?.(err);
    });
    return ctrl.abort.bind(ctrl);
  },
};

// Characters
export const characters = {
  list:         ()         => request('GET',    '/api/characters'),
  get:          (id)       => request('GET',    `/api/characters/${id}`),
  create:       (data)     => request('POST',   '/api/characters', data),
  update:       (id, data) => request('PATCH',  `/api/characters/${id}`, data),
  delete:       (id)       => request('DELETE', `/api/characters/${id}`),
  uploadAvatar: (id, file, crop) => uploadFile('PUT', `/api/characters/${id}/avatar${crop ? `?crop=${crop}` : ''}`, file),
  deleteAvatar: (id)       => request('DELETE', `/api/characters/${id}/avatar`),
  setDefault:   (id)       => request('POST',   `/api/characters/${id}/set-default`),
  clearDefault: ()         => request('DELETE', '/api/characters/default'),
};

// Completions
export const completions = {
  list: () => request('GET', '/api/completions'),
  get: (id) => request('GET', `/api/completions/${id}`),
  create: (model) => request('POST', '/api/completions', { model }),
  update: (id, data) => request('PATCH', `/api/completions/${id}`, data),
  delete: (id) => request('DELETE', `/api/completions/${id}`),
  regenerateTitle: (id) => request('POST', `/api/completions/${id}/regenerate-title`),
  undo: (id) => request('POST', `/api/completions/${id}/undo`),
  redo: (id) => request('POST', `/api/completions/${id}/redo`),
  run: (id, content, { onDelta, onDone, onError, maxTokens, temperature } = {}) => {
    const ctrl = new AbortController();
    const body = { content };
    if (maxTokens != null) body.max_tokens = maxTokens;
    if (temperature != null) body.temperature = temperature;
    fetch(`/api/completions/${id}/run`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
      body: JSON.stringify(body),
      signal: ctrl.signal,
    }).then(async (res) => {
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        onError?.(new Error(data.error || res.statusText));
        return;
      }
      await consumeSSE(res, (ev) => {
        const { delta, error } = ev;
        if (error) { onError?.(new Error(error)); return false; }
        if (delta) onDelta?.(delta);
      }, onDone);
    }).catch((err) => {
      if (err.name !== 'AbortError') onError?.(err);
    });
    return () => ctrl.abort();
  },
};

// Research
export const research = {
  list: () => request('GET', '/api/research'),
  defaults: () => request('GET', '/api/research/defaults'),
  get: (id) => request('GET', `/api/research/${id}`),
  createRemix: (id, model, direction, { onEvent, onDone, onError }) => {
    const ctrl = new AbortController();
    fetch(`/api/research/${id}/remixes`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
      body: JSON.stringify({ model, direction }),
      signal: ctrl.signal,
    }).then(async (res) => {
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        onError?.(new Error(data.error || res.statusText));
        return;
      }
      await consumeSSE(res, (ev) => { onEvent?.(ev); }, onDone);
    }).catch((err) => {
      if (err.name !== 'AbortError') onError?.(err);
    });
    return () => ctrl.abort();
  },
  getRemix: (id, remixId) => request('GET', `/api/research/${id}/remixes/${remixId}`),
  start: (title, query, model, mode, forceSearch, deepReport, pauseRedditImport, effort, maxTimeMinutes) =>
    request('POST', '/api/research', {
      title, query, ...(model ? { model } : {}), mode, force_search: forceSearch, deep_report: deepReport, pause_reddit_import: pauseRedditImport, effort, max_time_minutes: maxTimeMinutes,
    }),
  cancel: (id) => request('POST', `/api/research/${id}/cancel`),
  importReddit: (id, response) => request('POST', `/api/research/${id}/reddit-import`, response),
  skipReddit: (id, requestId) => request('POST', `/api/research/${id}/reddit-skip`, { request_id: requestId }),
  delete: (id) => request('DELETE', `/api/research/${id}`),
  // Streams progress events for a job. onEvent receives each parsed event;
  // a terminal event has a `status` field. Returns an abort() function.
  events: (id, { onEvent, onDone, onError }) => {
    const ctrl = new AbortController();
    fetch(`/api/research/${id}/events`, { signal: ctrl.signal }).then(async (res) => {
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        onError?.(new Error(data.error || res.statusText));
        return;
      }
      await consumeSSE(res, (ev) => { onEvent?.(ev); }, onDone);
    }).catch((err) => {
      if (err.name !== 'AbortError') onError?.(err);
    });
    return () => ctrl.abort();
  },
};

// Admin
export const admin = {
  users: {
    list: () => request('GET', '/api/admin/users'),
    create: (data) => request('POST', '/api/admin/users', data),
    update: (id, data) => request('PATCH', `/api/admin/users/${id}`, data),
    delete: (id) => request('DELETE', `/api/admin/users/${id}`),
  },
  tools: {
    listModels: () => request('GET', '/api/admin/tools/models'),
  },
  notePacks: {
    import: (pack) => request('POST', '/api/admin/note-packs/import', pack),
  },
};
