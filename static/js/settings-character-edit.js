import { auth, characters as charactersApi, models as modelsApi, tools as toolsApi } from './api.js';
import { preload as preloadIcons, icon } from './icons.js';
import { renderAvatarSection, attachAvatarSection } from './settings-avatar.js';
import { escapeHtml } from './utils.js';

let user = null;
let svgArrowLeft, svgUser, svgUsers, svgCpu, svgTrash, svgPlus, svgCopy;

const urlPath   = window.location.pathname;
const isNew     = urlPath === '/settings/characters/new';
const editMatch = urlPath.match(/^\/settings\/characters\/(\d+)\/edit$/);
const editId    = editMatch ? Number(editMatch[1]) : null;

// In-memory state for the hidden messages editor.
let hiddenMessages = [];
let dragSrcIndex   = null;

// In-memory state for the tools section.
let allTools    = [];
let enabledTools = [];

async function init() {
  try {
    user = await auth.me();
  } catch {
    window.location.href = '/';
    return;
  }

  if (!isNew && !editId) {
    window.location.href = '/settings/characters';
    return;
  }

  await preloadIcons();
  svgArrowLeft = icon('arrow-left', 14);
  svgUser      = icon('user', 16);
  svgUsers     = icon('users', 16);
  svgCpu       = icon('drama', 16);
  svgTrash     = icon('trash', 14);
  svgPlus      = icon('plus', 14);
  svgCopy      = icon('copy', 14);
  renderNav();

  let character  = null;
  let modelsData = [];

  try {
    if (isNew) {
      [modelsData, allTools] = await Promise.all([modelsApi.list(), toolsApi.list()]);
    } else {
      const [char, models, toolList] = await Promise.all([
        charactersApi.get(editId), modelsApi.list(), toolsApi.list(),
      ]);
      modelsData = models;
      allTools   = toolList;
      character  = char ?? null;
    }
  } catch {
    renderError('Could not load data.');
    return;
  }

  if (!isNew && !character) {
    renderError('Character not found.');
    return;
  }

  renderForm(character, modelsData);
}

function renderNav() {
  const adminSection = user.is_admin ? `
    <div class="snav-group-label">Admin</div>
    <a href="/settings/users" class="snav-item">
      ${svgUsers}
      Users
    </a>
    <a href="/settings/tools" class="snav-item">
      ${icon('sliders', 16)}
      Tools
    </a>
  ` : '';

  document.getElementById('snav').innerHTML = `
    <a href="/" class="snav-back">
      ${svgArrowLeft}
      Back to chat
    </a>
    <div class="snav-head">
      <img src="/assets/logo-mark.svg" alt="">
      <div class="title">Settings</div>
    </div>
    <div class="snav-group-label">You</div>
    <a href="/settings/account" class="snav-item">
      ${svgUser}
      Account
    </a>
    <a href="/settings/characters" class="snav-item active">
      ${svgCpu}
      Characters
    </a>
    ${adminSection}
  `;
}

function renderError(msg) {
  document.getElementById('smain').innerHTML = `
    <div class="smain-head">
      <h1>${isNew ? 'New character' : 'Edit character'}</h1>
    </div>
    <div class="section">
      <div class="field-msg field-msg--error" style="padding: 0 0 16px">${escapeHtml(msg)}</div>
      <a href="/settings/characters" class="btn btn-ghost btn-sm">${svgArrowLeft} Back to characters</a>
    </div>
  `;
}

function modelOptions(modelsData, selected) {
  return modelsData.map(m =>
    `<option value="${escapeHtml(m.name)}" ${m.name === selected ? 'selected' : ''}>${escapeHtml(m.display_name || m.name)}</option>`
  ).join('');
}

function renderForm(character, modelsData) {
  const isOwnerOrAdmin = !character || character.created_by === user.id || user.is_admin;
  const name         = character ? escapeHtml(character.name)         : '';
  const model        = character ? character.model                     : (modelsData[0]?.name ?? '');
  const systemPrompt = character ? (character.system_prompt ?? '')     : '';
  const firstMessage = character ? (character.first_message ?? '')     : '';
  const titlePrompt  = character ? (character.title_prompt  ?? '')     : '';
  const visibility   = character ? character.visibility                : 'private';
  const autoTitle    = character ? !!character.auto_title              : false;

  enabledTools = character?.tools ?? [];

  document.title = `${isNew ? 'New character' : 'Edit character'} — Settings — lemon chat`;

  document.getElementById('smain').innerHTML = `
    <div class="smain-head">
      <h1>${isNew ? 'New character' : 'Edit character'}</h1>
      ${!isNew ? `<p>${escapeHtml(character.name)}</p>` : ''}
    </div>
    <div class="section">
      <div class="character-form-fields">
        ${!isNew ? `<div class="character-form-row"><div id="char-avatar-section"></div></div>` : ''}
        <div class="character-form-row">
          <label class="character-form-lbl" for="char-name">Name</label>
          <input id="char-name" class="input" value="${name}" placeholder="Character name" autocomplete="off">
        </div>
        <div class="character-form-row">
          <label class="character-form-lbl" for="char-model">Model</label>
          <select id="char-model" class="input">${modelOptions(modelsData, model)}</select>
        </div>
        <div class="character-form-row">
          <label class="character-form-lbl" for="char-system-prompt">System prompt</label>
          <textarea id="char-system-prompt" class="input" rows="6" placeholder="Optional system prompt…">${escapeHtml(systemPrompt)}</textarea>
          <div class="character-form-hint">Template placeholders: <code>{{char}}</code> expands to the character's name, <code>{{user}}</code> to the user's name. Both work in the system prompt, hidden messages, and first message.</div>
        </div>
        <div class="character-form-row">
          <div class="character-form-lbl-row">
            <label class="character-form-lbl">Hidden messages</label>
            <button type="button" class="btn btn-ghost btn-sm" id="hidden-msg-add-btn">${svgPlus} Add message</button>
          </div>
          <div class="character-form-hint">Injected after the system prompt, before the conversation. Use these to prime the model with examples.</div>
          <div id="hidden-msgs-list"></div>
        </div>
        <div class="character-form-row">
          <label class="character-form-lbl" for="char-first-message">First message</label>
          <textarea id="char-first-message" class="input" rows="4" placeholder="Optional opening message…">${escapeHtml(firstMessage)}</textarea>
        </div>
        <div class="character-form-row character-form-check">
          <span class="character-form-lbl">Behaviour</span>
          <label id="char-auto-title-label">
            <div class="toggle${autoTitle ? ' on' : ''}" id="char-auto-title" role="checkbox" aria-checked="${autoTitle}" tabindex="0"></div>
            Generate title after first reply
          </label>
        </div>
        ${allTools.length > 0 ? `
        <div class="character-form-row">
          <span class="character-form-lbl">Tools</span>
          <div class="character-form-hint" style="margin-bottom:8px">Tools this character is allowed to call.</div>
          <div id="char-tools-list">
            ${allTools.map(t => `
              <label class="char-tool-row">
                <input type="checkbox" class="char-tool-checkbox" data-id="${escapeHtml(t.id)}" ${enabledTools.includes(t.id) ? 'checked' : ''}>
                <div class="char-tool-info">
                  <span class="char-tool-name">${escapeHtml(t.display_name)} <code class="char-tool-code">${escapeHtml(t.id)}</code></span>
                  <span class="char-tool-desc">${escapeHtml(t.description)}</span>
                </div>
              </label>
            `).join('')}
          </div>
        </div>` : ''}
        <div class="character-form-row">
          <label class="character-form-lbl" for="char-title-prompt">Title generation prompt</label>
          <textarea id="char-title-prompt" class="input" rows="3" placeholder="Leave empty to use the default prompt…">${escapeHtml(titlePrompt)}</textarea>
          <div class="character-form-hint">Default: "Generate a short title (at most 6 words) for the following conversation. Respond with only the title — no quotes, no trailing punctuation, no explanation."</div>
        </div>
        ${isOwnerOrAdmin ? `
        <div class="character-form-row">
          <label class="character-form-lbl" for="char-visibility">Visibility</label>
          <select id="char-visibility" class="input">
            <option value="private"   ${visibility === 'private'   ? 'selected' : ''}>Private — only you</option>
            <option value="readonly"  ${visibility === 'readonly'  ? 'selected' : ''}>Read-only — others can see, not edit</option>
            <option value="readwrite" ${visibility === 'readwrite' ? 'selected' : ''}>Read-write — others can see and edit</option>
          </select>
        </div>` : ''}
      </div>
      <div class="field-msg field-msg--error" id="char-err" hidden></div>
      <div class="char-form-actions">
        <a href="/settings/characters" class="btn btn-ghost btn-sm">Cancel</a>
        ${!isNew ? `<button class="btn btn-ghost btn-sm" id="char-clone-btn">${svgCopy} Clone</button>` : ''}
        <button class="btn btn-primary btn-sm" id="char-save-btn">${isNew ? 'Create character' : 'Save changes'}</button>
      </div>
    </div>
  `;

  hiddenMessages = (character?.hidden_messages ?? []).map(m => ({ role: m.role, content: m.content }));
  renderHiddenMessages();
  document.getElementById('hidden-msg-add-btn').addEventListener('click', addHiddenMessage);

  if (!isNew && character) {
    const avatarUrl = `/api/characters/${character.id}/avatar`;
    const charId = character.id;
    const avatarSection = document.getElementById('char-avatar-section');
    if (avatarSection) {
      avatarSection.innerHTML = renderAvatarSection(character.has_avatar, avatarUrl);
      attachAvatarSection(avatarSection, {
        avatarUrl,
        onUpload: async (file) => { await charactersApi.uploadAvatar(charId, file); },
        onDelete: async () => { await charactersApi.deleteAvatar(charId); },
      });
    }
  }

  const autoTitleEl = document.getElementById('char-auto-title');
  if (autoTitleEl) {
    const toggleAutoTitle = () => {
      autoTitleEl.classList.toggle('on');
      autoTitleEl.setAttribute('aria-checked', autoTitleEl.classList.contains('on'));
    };
    autoTitleEl.addEventListener('click', toggleAutoTitle);
    autoTitleEl.addEventListener('keydown', e => { if (e.key === ' ' || e.key === 'Enter') { e.preventDefault(); toggleAutoTitle(); } });
  }

  wireAutoExpand('char-system-prompt');
  wireAutoExpand('char-first-message');
  wireAutoExpand('char-title-prompt');

  document.getElementById('char-name').focus();
  document.getElementById('char-save-btn').addEventListener('click', () => {
    if (isNew) createChar();
    else saveChar(character);
  });
  document.getElementById('char-clone-btn')?.addEventListener('click', () => cloneChar(character));
}

function renderHiddenMessages() {
  const container = document.getElementById('hidden-msgs-list');
  if (!container) return;

  if (hiddenMessages.length === 0) {
    container.innerHTML = `<div class="hidden-msgs-empty">No hidden messages yet.</div>`;
    return;
  }

  container.innerHTML = hiddenMessages.map((msg, i) => `
    <div class="hidden-msg-row" data-index="${i}" draggable="true">
      <div class="hidden-msg-drag-handle" aria-hidden="true">⠿</div>
      <div class="hidden-msg-body">
        <select class="input hidden-msg-role" data-index="${i}">
          <option value="user"      ${msg.role === 'user'      ? 'selected' : ''}>user</option>
          <option value="assistant" ${msg.role === 'assistant' ? 'selected' : ''}>assistant</option>
        </select>
        <textarea class="input hidden-msg-content" data-index="${i}" rows="2">${escapeHtml(msg.content)}</textarea>
      </div>
      <button type="button" class="btn btn-ghost btn-sm hidden-msg-delete" data-index="${i}" aria-label="Remove">${svgTrash}</button>
    </div>
  `).join('');

  container.querySelectorAll('.hidden-msg-role').forEach(sel => {
    sel.addEventListener('change', e => {
      hiddenMessages[Number(e.target.dataset.index)].role = e.target.value;
    });
  });
  container.querySelectorAll('.hidden-msg-content').forEach(ta => {
    autoExpand(ta);
    ta.addEventListener('input', e => {
      hiddenMessages[Number(e.target.dataset.index)].content = e.target.value;
      autoExpand(e.target);
    });
  });
  container.querySelectorAll('.hidden-msg-delete').forEach(btn => {
    btn.addEventListener('click', e => {
      hiddenMessages.splice(Number(e.currentTarget.dataset.index), 1);
      renderHiddenMessages();
    });
  });

  container.querySelectorAll('.hidden-msg-row').forEach(row => {
    row.addEventListener('dragstart',  handleDragStart);
    row.addEventListener('dragover',   handleDragOver);
    row.addEventListener('dragleave',  handleDragLeave);
    row.addEventListener('drop',       handleDrop);
    row.addEventListener('dragend',    handleDragEnd);
  });
}

function addHiddenMessage() {
  hiddenMessages.push({ role: 'user', content: '' });
  renderHiddenMessages();
  const rows = document.querySelectorAll('.hidden-msg-content');
  rows[rows.length - 1]?.focus();
}

function handleDragStart(e) {
  dragSrcIndex = Number(e.currentTarget.dataset.index);
  e.dataTransfer.effectAllowed = 'move';
  requestAnimationFrame(() => e.currentTarget.classList.add('hidden-msg-row--dragging'));
}

function handleDragOver(e) {
  e.preventDefault();
  e.dataTransfer.dropEffect = 'move';
  const row  = e.currentTarget;
  const rect = row.getBoundingClientRect();
  const mid  = rect.top + rect.height / 2;
  row.classList.toggle('hidden-msg-row--drop-above', e.clientY < mid);
  row.classList.toggle('hidden-msg-row--drop-below', e.clientY >= mid);
}

function handleDragLeave(e) {
  if (e.currentTarget.contains(e.relatedTarget)) return;
  e.currentTarget.classList.remove('hidden-msg-row--drop-above', 'hidden-msg-row--drop-below');
}

function handleDrop(e) {
  e.preventDefault();
  const targetIndex = Number(e.currentTarget.dataset.index);
  if (dragSrcIndex === null || dragSrcIndex === targetIndex) {
    dragSrcIndex = null;
    renderHiddenMessages();
    return;
  }
  const rect    = e.currentTarget.getBoundingClientRect();
  const after   = e.clientY >= rect.top + rect.height / 2;
  const insertAt = after ? targetIndex + 1 : targetIndex;
  const [moved] = hiddenMessages.splice(dragSrcIndex, 1);
  const adjusted = dragSrcIndex < insertAt ? insertAt - 1 : insertAt;
  hiddenMessages.splice(adjusted, 0, moved);
  dragSrcIndex = null;
  renderHiddenMessages();
}

function handleDragEnd() {
  dragSrcIndex = null;
  document.querySelectorAll('.hidden-msg-row').forEach(r => {
    r.classList.remove('hidden-msg-row--dragging', 'hidden-msg-row--drop-above', 'hidden-msg-row--drop-below');
  });
}

function readForm(isOwnerOrAdmin) {
  const toolCheckboxes = document.querySelectorAll('.char-tool-checkbox');
  const tools = Array.from(toolCheckboxes)
    .filter(cb => cb.checked)
    .map(cb => cb.dataset.id);
  return {
    name:            document.getElementById('char-name')?.value.trim()          ?? '',
    model:           document.getElementById('char-model')?.value                ?? '',
    system_prompt:   document.getElementById('char-system-prompt')?.value.trim() || null,
    first_message:   document.getElementById('char-first-message')?.value.trim() || null,
    title_prompt:    document.getElementById('char-title-prompt')?.value.trim()  || null,
    visibility:      isOwnerOrAdmin ? (document.getElementById('char-visibility')?.value ?? undefined) : undefined,
    auto_title:      document.getElementById('char-auto-title')?.classList.contains('on') ?? false,
    tools,
    hidden_messages: hiddenMessages.map(m => ({ role: m.role, content: m.content })),
  };
}

async function createChar() {
  const data    = readForm(true);
  const errorEl = document.getElementById('char-err');
  const saveBtn = document.getElementById('char-save-btn');

  errorEl.hidden = true;
  if (!data.name)  { errorEl.hidden = false; errorEl.textContent = 'Name is required.';  return; }
  if (!data.model) { errorEl.hidden = false; errorEl.textContent = 'Model is required.'; return; }

  saveBtn.disabled    = true;
  saveBtn.textContent = 'Creating…';

  try {
    await charactersApi.create(data);
    window.location.href = '/settings/characters';
  } catch (err) {
    saveBtn.disabled    = false;
    saveBtn.textContent = 'Create character';
    errorEl.hidden      = false;
    errorEl.textContent = err.message;
  }
}

async function saveChar(character) {
  const isOwnerOrAdmin = character.created_by === user.id || user.is_admin;
  const data    = readForm(isOwnerOrAdmin);
  const errorEl = document.getElementById('char-err');
  const saveBtn = document.getElementById('char-save-btn');

  errorEl.hidden = true;
  if (!data.name)  { errorEl.hidden = false; errorEl.textContent = 'Name is required.';  return; }
  if (!data.model) { errorEl.hidden = false; errorEl.textContent = 'Model is required.'; return; }

  saveBtn.disabled    = true;
  saveBtn.textContent = 'Saving…';

  try {
    await charactersApi.update(character.id, data);
    window.location.href = '/settings/characters';
  } catch (err) {
    saveBtn.disabled    = false;
    saveBtn.textContent = 'Save changes';
    errorEl.hidden      = false;
    errorEl.textContent = err.message;
  }
}

async function cloneChar(character) {
  const errorEl   = document.getElementById('char-err');
  const cloneBtn  = document.getElementById('char-clone-btn');

  errorEl.hidden     = true;
  cloneBtn.disabled  = true;
  cloneBtn.textContent = 'Cloning…';

  try {
    const data = {
      name:            `Copy of ${character.name}`,
      model:           character.model,
      system_prompt:   character.system_prompt ?? null,
      first_message:   character.first_message ?? null,
      title_prompt:    character.title_prompt  ?? null,
      visibility:      'private',
      auto_title:      character.auto_title,
      tools:           character.tools ?? [],
      hidden_messages: hiddenMessages.map(m => ({ role: m.role, content: m.content })),
    };
    const created = await charactersApi.create(data);
    window.location.href = `/settings/characters/${created.id}/edit`;
  } catch (err) {
    cloneBtn.disabled    = false;
    cloneBtn.innerHTML   = `${svgCopy} Clone`;
    errorEl.hidden       = false;
    errorEl.textContent  = err.message;
  }
}

function autoExpand(el) {
  el.style.overflow = 'hidden';
  el.style.resize   = 'none';
  el.style.height   = 'auto';
  el.style.height   = el.scrollHeight + 'px';
}

function wireAutoExpand(id) {
  const el = document.getElementById(id);
  if (!el) return;
  autoExpand(el);
  el.addEventListener('input', () => autoExpand(el));
}


init();
