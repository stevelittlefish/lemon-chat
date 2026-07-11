import { admin } from './api.js';
import { requireAuth } from './settings-auth.js';
import { preload as preloadIcons, icon } from './icons.js';
import { escapeHtml } from './utils.js';

let user = null;
let svgArrowLeft, svgUser, svgUsers, svgSliders, svgCopy;

async function init() {
  user = await requireAuth();
  if (!user) return;
  if (!user.is_admin) {
    window.location.href = '/settings/account';
    return;
  }
  await preloadIcons();
  svgArrowLeft = icon('arrow-left', 14);
  svgUser      = icon('user', 16);
  svgUsers     = icon('users', 16);
  svgSliders   = icon('sliders', 16);
  svgCopy      = icon('copy', 14);
  renderNav();
  renderPage();
}

// ── Nav ───────────────────────────────────────────────────────────────

function renderNav() {
  const path = window.location.pathname;

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
    <a href="/settings/notes" class="snav-item${path === '/settings/notes' ? ' active' : ''}">
      ${icon('file-text', 16)}
      Notes
    </a>
    <div class="snav-group-label">Admin</div>
    <a href="/settings/users" class="snav-item${path === '/settings/users' ? ' active' : ''}">
      ${svgUsers}
      Users
    </a>
    <a href="/settings/tools" class="snav-item${path === '/settings/tools' ? ' active' : ''}">
      ${svgSliders}
      Tools
    </a>
  `;
}

// ── Tools page ────────────────────────────────────────────────────────

function renderPage() {
  document.getElementById('smain').innerHTML = `
    <div class="smain-head">
      <h1>Tools</h1>
      <p>Imports, configuration helpers and diagnostics.</p>
    </div>
    <div class="section">
      <h2>Models</h2>
      <div class="tool-row">
        <div class="tool-row-info">
          <div class="tool-row-name">List provider models</div>
          <div class="tool-row-desc">Query every configured provider for model IDs you can add to lemon.toml.</div>
        </div>
        <button class="btn btn-secondary btn-sm" id="btn-list-models">List models</button>
      </div>
      <div id="provider-models-result" class="provider-models hidden"></div>
    </div>
    <div class="section">
      <h2>Conversations</h2>
      <div class="tool-row">
        <div class="tool-row-info">
          <div class="tool-row-name">Import chat</div>
          <div class="tool-row-desc">Recreate a conversation from a JSON export as a new conversation on your account.</div>
        </div>
        <a href="/settings/import_chat" class="btn btn-secondary btn-sm">Import</a>
      </div>
    </div>
    <div class="section">
      <h2>Note packs</h2>
      <div class="tool-row">
        <div class="tool-row-info">
          <div class="tool-row-name">Import note pack</div>
          <div class="tool-row-desc">Load a .json note pack — creates or updates all notes in the pack. Existing notes with the same key are overwritten.</div>
        </div>
        <div>
          <input type="file" id="note-pack-file" accept=".json" style="display:none">
          <button class="btn btn-secondary btn-sm" id="btn-import-note-pack">Choose file</button>
        </div>
      </div>
      <div id="note-pack-result" class="tool-result hidden"></div>
    </div>
  `;

  document.getElementById('btn-import-note-pack').addEventListener('click', () => {
    document.getElementById('note-pack-file').click();
  });
  document.getElementById('note-pack-file').addEventListener('change', runImportNotePack);
  document.getElementById('btn-list-models').addEventListener('click', runListModels);
}

async function runListModels() {
  const btn = document.getElementById('btn-list-models');
  const result = document.getElementById('provider-models-result');

  btn.disabled = true;
  btn.textContent = 'Loading…';
  result.className = 'provider-models';
  result.innerHTML = '<div class="provider-models-status">Querying providers…</div>';

  try {
    const providers = await admin.tools.listModels();
    result.innerHTML = providers.length === 0
      ? '<div class="provider-models-status">No model providers are configured.</div>'
      : providers.map(renderProviderModels).join('');
    result.querySelectorAll('[data-copy-model]').forEach(copyBtn => {
      copyBtn.addEventListener('click', () => copyModelID(copyBtn));
    });
  } catch (err) {
    result.innerHTML = `<div class="tool-result tool-result--error">Failed: ${escapeHtml(err.message || 'unknown error')}</div>`;
  } finally {
    btn.disabled = false;
    btn.textContent = 'Refresh';
  }
}

function renderProviderModels(provider) {
  let body;
  if (provider.error) {
    body = `<div class="provider-error">Could not list models: ${escapeHtml(provider.error)}</div>`;
  } else if (provider.models.length === 0) {
    body = '<div class="provider-empty">This provider returned no models.</div>';
  } else {
    body = `
      <div class="provider-model-table-wrap">
        <table class="provider-model-table">
          <thead><tr><th>Model ID</th><th><span class="sr-only">Copy</span></th></tr></thead>
          <tbody>
            ${provider.models.map(model => `
              <tr>
                <td><code class="provider-model-id">${escapeHtml(model)}</code></td>
                <td class="provider-model-action">
                  <button class="btn btn-ghost btn-sm provider-copy" data-copy-model="${escapeHtml(model)}" aria-label="Copy ${escapeHtml(model)}">
                    ${svgCopy}<span>Copy</span>
                  </button>
                </td>
              </tr>
            `).join('')}
          </tbody>
        </table>
      </div>`;
  }

  return `
    <section class="provider-models-group">
      <div class="provider-models-head">
        <h3>${escapeHtml(provider.name)}</h3>
        <code>${escapeHtml(provider.api_base)}</code>
      </div>
      ${body}
    </section>`;
}

async function copyModelID(btn) {
  const model = btn.dataset.copyModel;
  try {
    await navigator.clipboard.writeText(model);
    btn.querySelector('span').textContent = 'Copied';
    window.setTimeout(() => { btn.querySelector('span').textContent = 'Copy'; }, 1200);
  } catch {
    const range = document.createRange();
    range.selectNodeContents(btn.closest('tr').querySelector('.provider-model-id'));
    const selection = window.getSelection();
    selection.removeAllRanges();
    selection.addRange(range);
  }
}

async function runImportNotePack(e) {
  const file   = e.target.files[0];
  const btn    = document.getElementById('btn-import-note-pack');
  const result = document.getElementById('note-pack-result');
  if (!file) return;

  e.target.value = '';
  btn.disabled    = true;
  btn.textContent = 'Importing…';
  result.className = 'tool-result hidden';

  let pack;
  try {
    pack = JSON.parse(await file.text());
  } catch {
    result.className   = 'tool-result tool-result--error';
    result.textContent = 'Failed: could not parse JSON file.';
    result.classList.remove('hidden');
    btn.disabled    = false;
    btn.textContent = 'Choose file';
    return;
  }

  try {
    const data = await admin.notePacks.import(pack);
    result.className   = 'tool-result tool-result--ok';
    result.textContent = `Imported ${data.imported} note${data.imported === 1 ? '' : 's'} from "${data.pack_name}" v${data.pack_version}.`;
  } catch (err) {
    result.className   = 'tool-result tool-result--error';
    result.textContent = 'Failed: ' + (err.message || 'unknown error');
  } finally {
    btn.disabled    = false;
    btn.textContent = 'Choose file';
    result.classList.remove('hidden');
  }
}

init();
