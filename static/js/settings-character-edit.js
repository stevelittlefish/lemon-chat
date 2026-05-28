import { auth, characters as charactersApi, models as modelsApi } from './api.js';
import { preload as preloadIcons, icon } from './icons.js';

let user = null;
let svgArrowLeft, svgUser, svgUsers, svgCpu;

const urlPath   = window.location.pathname;
const isNew     = urlPath === '/settings/characters/new';
const editMatch = urlPath.match(/^\/settings\/characters\/(\d+)\/edit$/);
const editId    = editMatch ? Number(editMatch[1]) : null;

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
  renderNav();

  let character  = null;
  let modelsData = [];

  try {
    if (isNew) {
      modelsData = await modelsApi.list();
    } else {
      const [chars, models] = await Promise.all([charactersApi.list(), modelsApi.list()]);
      modelsData = models;
      character  = chars.find(c => c.id === editId) ?? null;
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
  const visibility   = character ? character.visibility                : 'private';

  document.title = `${isNew ? 'New character' : 'Edit character'} — Settings — lemon chat`;

  document.getElementById('smain').innerHTML = `
    <div class="smain-head">
      <h1>${isNew ? 'New character' : 'Edit character'}</h1>
      ${!isNew ? `<p>${escapeHtml(character.name)}</p>` : ''}
    </div>
    <div class="section">
      <div class="character-form-fields">
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
        </div>
        <div class="character-form-row">
          <label class="character-form-lbl" for="char-first-message">First message</label>
          <textarea id="char-first-message" class="input" rows="4" placeholder="Optional opening message…">${escapeHtml(firstMessage)}</textarea>
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
        <button class="btn btn-primary btn-sm" id="char-save-btn">${isNew ? 'Create character' : 'Save changes'}</button>
      </div>
    </div>
  `;

  document.getElementById('char-name').focus();
  document.getElementById('char-save-btn').addEventListener('click', () => {
    if (isNew) createChar();
    else saveChar(character);
  });
}

function readForm(isOwnerOrAdmin) {
  return {
    name:          document.getElementById('char-name')?.value.trim()          ?? '',
    model:         document.getElementById('char-model')?.value                ?? '',
    system_prompt: document.getElementById('char-system-prompt')?.value.trim() || null,
    first_message: document.getElementById('char-first-message')?.value.trim() || null,
    visibility:    isOwnerOrAdmin ? (document.getElementById('char-visibility')?.value ?? undefined) : undefined,
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

function escapeHtml(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

init();
