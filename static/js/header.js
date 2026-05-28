import { icon } from './icons.js';

const headerEl = document.getElementById('chat-header');

let state = {
  conversationId: null,
  title: null,
  models: [],
  characters: [],
  // { type: 'model', name: string } | { type: 'character', id: number }
  selection: null,
};

export function init({ models, characters }) {
  state.models = models;
  state.characters = characters;
  const def = models.find(m => m.default) ?? models[0];
  state.selection = def ? { type: 'model', name: def.name } : null;
  render();
}

export function setConversation(id, title) {
  state.conversationId = id;
  state.title = title ?? null;
  render();
}

export function updateTitle(title) {
  state.title = title;
  const el = headerEl.querySelector('.chat-header-title');
  if (el) el.textContent = title || '';
}

// Sync picker to the loaded conversation's model/character.
export function setSelection(conv) {
  if (!conv) return;
  if (conv.character_id != null) {
    state.selection = { type: 'character', id: conv.character_id };
  } else if (conv.model != null) {
    state.selection = { type: 'model', name: conv.model };
  }
  const label = document.getElementById('model-label');
  if (label) label.textContent = selectedDisplayName();
}

// Returns { type: 'model', name } or { type: 'character', id }.
export function getSelection() {
  return state.selection;
}

function render() {
  const hasConv = state.conversationId !== null;
  headerEl.innerHTML = `
    <div class="chat-header-title-wrap">
      ${hasConv
        ? `<span class="chat-header-title">${escapeHtml(state.title || '')}</span>`
        : `<span class="chat-header-title chat-header-title--wordmark">lemon chat</span>`
      }
    </div>
    <div class="model-picker-wrap" id="model-picker-wrap">
      <button class="model-picker-trigger" id="model-picker-trigger" type="button">
        ${icon('cpu', 13)}
        <span id="model-label">${escapeHtml(selectedDisplayName())}</span>
        ${icon('chevron-down', 12)}
      </button>
    </div>
  `;

  document.getElementById('model-picker-trigger').addEventListener('click', (e) => {
    e.stopPropagation();
    toggleDropdown();
  });
}

function toggleDropdown() {
  const existing = document.getElementById('model-picker-dropdown');
  if (existing) { existing.remove(); return; }

  const wrap = document.getElementById('model-picker-wrap');
  const dropdown = document.createElement('div');
  dropdown.id = 'model-picker-dropdown';
  dropdown.className = 'model-picker-dropdown';

  const isMSel = state.selection?.type === 'model';
  const isCSel = state.selection?.type === 'character';

  let html = `<div class="model-picker-dropdown-label">model</div>`;
  html += state.models.map(m => {
    const sel = isMSel && m.name === state.selection.name;
    return `
      <div class="model-picker-option${sel ? ' selected' : ''}" data-type="model" data-name="${escapeHtml(m.name)}">
        <span class="model-picker-option-check">${sel ? icon('check', 13) : ''}</span>
        <span class="model-picker-option-name">${escapeHtml(m.display_name)}</span>
      </div>`;
  }).join('');

  if (state.characters.length) {
    html += `<div class="model-picker-divider"></div>`;
    html += `<div class="model-picker-dropdown-label">character</div>`;
    html += state.characters.map(c => {
      const sel = isCSel && c.id === state.selection.id;
      return `
        <div class="model-picker-option${sel ? ' selected' : ''}" data-type="character" data-id="${c.id}">
          <span class="model-picker-option-check">${sel ? icon('check', 13) : ''}</span>
          <span class="model-picker-option-name">${escapeHtml(c.name)}</span>
        </div>`;
    }).join('');
  }

  dropdown.innerHTML = html;

  dropdown.querySelectorAll('.model-picker-option').forEach(opt => {
    opt.addEventListener('click', () => {
      if (opt.dataset.type === 'model') {
        state.selection = { type: 'model', name: opt.dataset.name };
      } else {
        state.selection = { type: 'character', id: Number(opt.dataset.id) };
      }
      const label = document.getElementById('model-label');
      if (label) label.textContent = selectedDisplayName();
      dropdown.remove();
    });
  });

  wrap.appendChild(dropdown);

  setTimeout(() => {
    document.addEventListener('click', () => {
      document.getElementById('model-picker-dropdown')?.remove();
    }, { once: true });
  }, 0);
}

function selectedDisplayName() {
  if (!state.selection) return 'no model';
  if (state.selection.type === 'model') {
    return state.models.find(m => m.name === state.selection.name)?.display_name ?? state.selection.name;
  }
  return state.characters.find(c => c.id === state.selection.id)?.name ?? 'character';
}

function escapeHtml(str) {
  return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
