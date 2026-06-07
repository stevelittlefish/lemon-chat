import { characters as charactersApi, models as modelsApi } from './api.js';
import { requireAuth } from './settings-auth.js';
import { preload as preloadIcons, icon } from './icons.js';
import { escapeHtml } from './utils.js';

let user = null;
let svgArrowLeft, svgUser, svgUsers, svgCpu, svgPlus, svgPencil, svgTrash, svgDownload, svgUpload;

let charactersData = [];

async function init() {
  user = await requireAuth();
  if (!user) return;
  await preloadIcons();
  svgArrowLeft = icon('arrow-left', 14);
  svgUser      = icon('user', 16);
  svgUsers     = icon('users', 16);
  svgCpu       = icon('drama', 16);
  svgPlus      = icon('plus', 14);
  svgPencil    = icon('pencil', 13);
  svgTrash     = icon('trash', 13);
  svgDownload  = icon('download', 13);
  svgUpload    = icon('upload', 14);
  renderNav();
  renderPage();
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
      ${svgCpu}
      Characters
    </a>
    ${adminSection}
  `;
}

// ── Characters page ───────────────────────────────────────────────────

async function renderPage() {
  document.getElementById('smain').innerHTML = `
    <div class="smain-head">
      <h1>Characters</h1>
      <p>Reusable AI personas with a model, system prompt, and optional opening message.</p>
    </div>
    <div class="section">
      <div class="section-toolbar">
        <div>
          <h2>My characters</h2>
          <p class="lead">Characters you created.</p>
        </div>
        <div class="toolbar-actions">
          <button class="btn btn-secondary btn-sm" id="import-char-btn">
            ${svgUpload}
            Import
          </button>
          <input type="file" id="import-char-input" accept=".json" style="display:none">
          <button class="btn btn-secondary btn-sm" id="import-card-btn">
            ${svgUpload}
            Import character card
          </button>
          <input type="file" id="import-card-input" accept=".png" style="display:none">
          <a href="/settings/characters/new" class="btn btn-secondary btn-sm">
            ${svgPlus}
            Add character
          </a>
        </div>
      </div>
      <table class="chars-table">
        <thead><tr><th>Name</th><th>Model</th><th>Visibility</th><th></th></tr></thead>
        <tbody id="my-chars-list"><tr><td colspan="4" class="users-status">Loading…</td></tr></tbody>
      </table>
    </div>
    <div class="section" id="other-chars-section">
      <h2>Other characters</h2>
      <p class="lead">Characters created by other users.</p>
      <table class="chars-table">
        <thead><tr><th>Name</th><th>Owner</th><th>Model</th><th>Visibility</th><th></th></tr></thead>
        <tbody id="other-chars-list"><tr><td colspan="5" class="users-status">Loading…</td></tr></tbody>
      </table>
    </div>
  `;

  try {
    charactersData = await charactersApi.list();
  } catch {
    const err = `<div class="field-msg field-msg--error" style="padding:14px 0">Could not load characters.</div>`;
    document.getElementById('my-chars-list').innerHTML = err;
    document.getElementById('other-chars-list').innerHTML = err;
    return;
  }
  renderCharacterLists();
}

function canEditChar(c) {
  return c.created_by === user.id || user.is_admin || c.visibility === 'readwrite';
}

function canDeleteChar(c) {
  return c.created_by === user.id || user.is_admin;
}

function renderCharacterLists() {
  const myList       = document.getElementById('my-chars-list');
  const otherList    = document.getElementById('other-chars-list');
  const otherSection = document.getElementById('other-chars-section');
  if (!myList) return;

  const mine  = charactersData.filter(c => c.created_by === user.id);
  const other = charactersData.filter(c => c.created_by !== user.id);

  myList.innerHTML    = renderCharRows(mine,  'You have no characters yet.', false);
  otherList.innerHTML = renderCharRows(other, 'No characters from other users.', true);
  if (otherSection) otherSection.hidden = other.length === 0;

  attachRowEvents(myList);
  attachRowEvents(otherList);

  const importBtn   = document.getElementById('import-char-btn');
  const importInput = document.getElementById('import-char-input');
  if (importBtn && importInput) {
    importBtn.addEventListener('click', () => importInput.click());
    importInput.addEventListener('change', () => {
      if (importInput.files[0]) importChar(importInput.files[0]);
    });
  }

  const importCardBtn   = document.getElementById('import-card-btn');
  const importCardInput = document.getElementById('import-card-input');
  if (importCardBtn && importCardInput) {
    importCardBtn.addEventListener('click', () => importCardInput.click());
    importCardInput.addEventListener('change', () => {
      if (importCardInput.files[0]) importSillyTavernCard(importCardInput.files[0]);
    });
  }
}

function renderCharRows(list, emptyMsg, showOwner) {
  const colspan = showOwner ? 5 : 4;
  if (!list.length) {
    return `<tr><td colspan="${colspan}" class="users-status">${emptyMsg}</td></tr>`;
  }
  return list.map(c => {
    const exportBtn = `<button class="btn btn-ghost btn-sm" data-action="export" data-id="${c.id}">${svgDownload} Export</button>`;
    const editBtn   = canEditChar(c)   ? `<a href="/settings/characters/${c.id}/edit" class="btn btn-ghost btn-sm">${svgPencil} Edit</a>` : '';
    const deleteBtn = canDeleteChar(c) ? `<button class="btn btn-ghost btn-sm" data-action="delete" data-id="${c.id}">${svgTrash} Delete</button>` : '';
    const visLabel  = { private: 'private', readonly: 'read-only', readwrite: 'read-write' }[c.visibility] ?? c.visibility;
    const avatarHtml = c.has_avatar
      ? `<img class="avatar-sm" src="/api/characters/${c.id}/avatar" alt="">`
      : '';
    const starHtml = c.is_default ? `<img src="/assets/icons/star-mustard.svg" class="char-list-star" width="12" height="12" alt="" title="Default character">` : '';
    const ownerCell = showOwner ? `<td class="char-col-owner">${escapeHtml(c.created_by_name)}</td>` : '';
    return `
      <tr data-id="${c.id}">
        <td class="char-col-name"><div class="char-name-cell">${starHtml}${avatarHtml}${escapeHtml(c.name)}</div></td>
        ${ownerCell}
        <td class="char-col-model"><span class="chip">${escapeHtml(c.model)}</span></td>
        <td class="char-col-visibility"><span class="chip character-chip--${c.visibility}">${visLabel}</span></td>
        <td class="char-col-actions"><div class="user-row-actions">${exportBtn}${editBtn}${deleteBtn}</div></td>
      </tr>`;
  }).join('');
}

function attachRowEvents(container) {
  container.querySelectorAll('[data-action="delete"]').forEach(btn => {
    const id = Number(btn.dataset.id);
    btn.addEventListener('click', () => deleteChar(id));
  });
  container.querySelectorAll('[data-action="export"]').forEach(btn => {
    const id = Number(btn.dataset.id);
    btn.addEventListener('click', () => exportChar(id));
  });
}

async function deleteChar(id) {
  const c = charactersData.find(c => c.id === id);
  if (!confirm(`Delete character "${c?.name}"? This cannot be undone.`)) return;

  try {
    await charactersApi.delete(id);
    charactersData = charactersData.filter(c => c.id !== id);
    renderCharacterLists();
  } catch (err) {
    alert(`Could not delete character: ${err.message}`);
  }
}

// ── Export / Import ───────────────────────────────────────────────────

async function exportChar(id) {
  let data;
  try {
    data = await charactersApi.get(id);
  } catch (err) {
    alert(`Could not export character: ${err.message}`);
    return;
  }
  const payload = {
    name:            data.name,
    model:           data.model,
    system_prompt:   data.system_prompt,
    first_message:   data.first_message,
    title_prompt:    data.title_prompt,
    visibility:      data.visibility,
    auto_title:      data.auto_title,
    hidden_messages: (data.hidden_messages || []).map(m => ({
      role:       m.role,
      content:    m.content,
      sort_order: m.sort_order,
    })),
  };

  // Include avatar as base64 data URL if the character has one.
  if (data.has_avatar) {
    try {
      const res = await fetch(`/api/characters/${id}/avatar`);
      if (res.ok) {
        const ab = await res.arrayBuffer();
        const ct = res.headers.get('content-type') || 'image/jpeg';
        const b64 = btoa(String.fromCharCode(...new Uint8Array(ab)));
        payload.avatar = `data:${ct};base64,${b64}`;
      }
    } catch { /* skip avatar on error */ }
  }

  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' });
  const url  = URL.createObjectURL(blob);
  const a    = document.createElement('a');
  a.href     = url;
  a.download = data.name.replace(/[^a-z0-9_-]/gi, '_').toLowerCase() + '.json';
  a.click();
  URL.revokeObjectURL(url);
}

async function importChar(file) {
  let parsed;
  try {
    parsed = JSON.parse(await file.text());
  } catch {
    alert('Invalid JSON file.');
    return;
  }
  if (!parsed.name || !parsed.model) {
    alert('Invalid character file: name and model are required.');
    return;
  }
  try {
    const created = await charactersApi.create(parsed);
    // Restore avatar from base64 data URL if present.
    if (parsed.avatar && created?.id) {
      try {
        const [header, b64] = parsed.avatar.split(',');
        const mime = (header.match(/:(.*?);/) || [])[1] || 'image/jpeg';
        const ext  = mime.split('/')[1]?.replace('jpeg', 'jpg') || 'jpg';
        const bytes = Uint8Array.from(atob(b64), c => c.charCodeAt(0));
        const blob  = new Blob([bytes], { type: mime });
        const avatarFile = new File([blob], `avatar.${ext}`, { type: mime });
        await charactersApi.uploadAvatar(created.id, avatarFile);
      } catch { /* skip avatar on error */ }
    }
    await renderPage();
  } catch (err) {
    alert(`Could not import character: ${err.message}`);
  }
}


// ── SillyTavern character card import ────────────────────────────────

function parseSillyTavernPng(buffer) {
  const view = new DataView(buffer);
  let pos = 8; // skip PNG signature
  while (pos < buffer.byteLength) {
    const length    = view.getUint32(pos, false);
    const chunkType = String.fromCharCode(
      view.getUint8(pos + 4), view.getUint8(pos + 5),
      view.getUint8(pos + 6), view.getUint8(pos + 7),
    );
    if (chunkType === 'tEXt') {
      const bytes = new Uint8Array(buffer, pos + 8, length);
      const nullIdx = bytes.indexOf(0);
      if (nullIdx !== -1) {
        const keyword = new TextDecoder('latin1').decode(bytes.slice(0, nullIdx));
        if (keyword === 'chara') {
          const value = new TextDecoder('latin1').decode(bytes.slice(nullIdx + 1));
          const bin = atob(value.trim());
          const utf8 = new Uint8Array(bin.length);
          for (let i = 0; i < bin.length; i++) utf8[i] = bin.charCodeAt(i);
          return JSON.parse(new TextDecoder('utf-8').decode(utf8));
        }
      }
    }
    pos += 12 + length;
  }
  throw new Error('No chara chunk found in PNG.');
}

function parseMesExample(text) {
  if (!text || !text.trim()) return [];
  const messages = [];
  let order = 0;
  for (const block of text.split(/<START>/i)) {
    for (const line of block.split(/\r?\n/)) {
      const userMatch = line.match(/^\{\{user\}\}:\s*/i);
      const charMatch = line.match(/^\{\{char\}\}:\s*/i);
      if (userMatch) {
        messages.push({ role: 'user', content: line.slice(userMatch[0].length).trim(), sort_order: order++ });
      } else if (charMatch) {
        messages.push({ role: 'assistant', content: line.slice(charMatch[0].length).trim(), sort_order: order++ });
      }
    }
  }
  return messages.filter(m => m.content);
}

function buildCharFromCard(card, defaultModel) {
  const d = card.data || {};

  const name        = (d.name        || card.name        || '').trim();
  const systemP     = (d.system_prompt                   || '').trim();
  const description = (d.description || card.description || '').trim();
  const personality = (d.personality || card.personality || '').trim();
  const scenario    = (d.scenario    || card.scenario    || '').trim();
  const firstMes    = (d.first_mes   || card.first_mes   || '').trim();
  const mesExample  = (d.mes_example || card.mes_example || '').trim();

  const systemParts = [systemP, description, personality, scenario].filter(Boolean);
  const system_prompt  = systemParts.join('\n\n') || null;
  const first_message  = firstMes || null;
  const hidden_messages = parseMesExample(mesExample);

  return {
    name,
    model:          defaultModel,
    system_prompt,
    first_message,
    visibility:     'private',
    auto_title:     false,
    hidden_messages,
  };
}

async function importSillyTavernCard(file) {
  let card;
  try {
    const buffer = await file.arrayBuffer();
    card = parseSillyTavernPng(buffer);
  } catch (err) {
    alert(`Could not read character card: ${err.message}`);
    return;
  }

  let defaultModel = '';
  try {
    const modelList = await modelsApi.list();
    defaultModel = modelList[0]?.name ?? '';
  } catch { /* leave blank; user can edit */ }

  const payload = buildCharFromCard(card, defaultModel);
  if (!payload.name) {
    alert('Character card has no name.');
    return;
  }

  try {
    const created = await charactersApi.create(payload);
    if (created?.id) {
      try {
        await charactersApi.uploadAvatar(created.id, file, 'top');
      } catch { /* skip avatar on error */ }
      window.location.href = `/settings/characters/${created.id}/edit`;
    }
  } catch (err) {
    alert(`Could not import character card: ${err.message}`);
  }
}

init();
