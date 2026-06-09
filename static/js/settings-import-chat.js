import { conversations as convsApi, models as modelsApi, characters as charsApi } from './api.js';
import { requireAuth } from './settings-auth.js';
import { preload as preloadIcons, icon } from './icons.js';
import { escapeHtml } from './utils.js';

let user = null;
let svgArrowLeft, svgUser, svgUsers;

async function init() {
  user = await requireAuth();
  if (!user) return;
  await preloadIcons();
  svgArrowLeft = icon('arrow-left', 14);
  svgUser      = icon('user', 16);
  svgUsers     = icon('users', 16);
  renderNav();
  await renderPage();
}

// ── Nav ───────────────────────────────────────────────────────────────

function renderNav() {
  const path = window.location.pathname;
  const adminSection = user.is_admin ? `
    <div class="snav-group-label">Admin</div>
    <a href="/settings/users" class="snav-item${path === '/settings/users' ? ' active' : ''}">
      ${svgUsers}
      Users
    </a>
    <a href="/settings/tools" class="snav-item${path === '/settings/tools' ? ' active' : ''}">
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
    <a href="/settings/account" class="snav-item${path === '/settings/account' ? ' active' : ''}">
      ${svgUser}
      Account
    </a>
    <a href="/settings/characters" class="snav-item${path === '/settings/characters' ? ' active' : ''}">
      ${icon('drama', 16)}
      Characters
    </a>
    ${adminSection}
  `;
}

// ── Import page ───────────────────────────────────────────────────────

async function renderPage() {
  let models = [];
  let chars  = [];
  try {
    [models, chars] = await Promise.all([modelsApi.list('chat'), charsApi.list()]);
  } catch (err) {
    document.getElementById('smain').innerHTML = `
      <div class="smain-head"><h1>Import chat</h1></div>
      <div class="section"><p class="tool-result tool-result--error">Failed to load models: ${escapeHtml(err.message)}</p></div>
    `;
    return;
  }

  const modelOptions = models.map(m =>
    `<option value="model:${escapeHtml(m.name)}">${escapeHtml(m.display_name || m.name)}</option>`
  ).join('');

  const charOptions = chars.map(c =>
    `<option value="character:${c.id}">${escapeHtml(c.name)}</option>`
  ).join('');

  const groupedOptions = [
    models.length ? `<optgroup label="Models">${modelOptions}</optgroup>` : '',
    chars.length  ? `<optgroup label="Characters">${charOptions}</optgroup>` : '',
  ].join('');

  document.getElementById('smain').innerHTML = `
    <div class="smain-head">
      <h1>Import chat</h1>
      <p>Recreate a conversation from a JSON export. Messages are imported in order; system prompts are skipped.</p>
    </div>
    <div class="section">
      <div class="character-form-fields">
        <div class="character-form-row">
          <label class="character-form-lbl" for="import-file">JSON file</label>
          <input id="import-file" type="file" accept=".json,application/json" class="input">
        </div>
        <div class="character-form-row">
          <label class="character-form-lbl" for="import-model">Model or character</label>
          <select id="import-model" class="input">
            <option value="">Select…</option>
            ${groupedOptions}
          </select>
        </div>
      </div>
      <div style="margin-top: 20px;">
        <button id="import-btn" class="btn btn-primary" disabled>Import</button>
      </div>
      <div id="import-result" class="tool-result hidden" style="margin-top: 12px;"></div>
    </div>
  `;

  const fileInput   = document.getElementById('import-file');
  const modelSelect = document.getElementById('import-model');
  const importBtn   = document.getElementById('import-btn');
  const resultEl    = document.getElementById('import-result');

  let parsedMessages = null;

  function updateButton() {
    importBtn.disabled = !(parsedMessages && modelSelect.value);
  }

  fileInput.addEventListener('change', () => {
    parsedMessages = null;
    resultEl.className = 'tool-result hidden';
    const file = fileInput.files[0];
    if (!file) { updateButton(); return; }

    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const data = JSON.parse(e.target.result);
        if (!Array.isArray(data)) throw new Error('File must contain a JSON array.');
        parsedMessages = data;
        updateButton();
      } catch (err) {
        resultEl.className   = 'tool-result tool-result--error';
        resultEl.textContent = 'Invalid JSON: ' + err.message;
        resultEl.classList.remove('hidden');
        updateButton();
      }
    };
    reader.readAsText(file);
  });

  modelSelect.addEventListener('change', updateButton);

  importBtn.addEventListener('click', async () => {
    if (!parsedMessages || !modelSelect.value) return;

    const [type, value] = modelSelect.value.split(':');
    const payload = { messages: parsedMessages };
    if (type === 'model') {
      payload.model = value;
    } else {
      payload.character_id = parseInt(value, 10);
    }

    importBtn.disabled    = true;
    importBtn.textContent = 'Importing…';
    resultEl.className    = 'tool-result hidden';

    try {
      const conv = await convsApi.importChat(payload);
      resultEl.className   = 'tool-result tool-result--ok';
      resultEl.innerHTML   = `Conversation imported. <a href="/?c=${conv.id}">Open conversation</a>`;
    } catch (err) {
      resultEl.className   = 'tool-result tool-result--error';
      resultEl.textContent = 'Import failed: ' + (err.message || 'unknown error');
    } finally {
      importBtn.disabled    = false;
      importBtn.textContent = 'Import';
      resultEl.classList.remove('hidden');
    }
  });
}

init();
