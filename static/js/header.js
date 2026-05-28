import { icon } from './icons.js';

const headerEl = document.getElementById('chat-header');

let state = {
  conversationId: null,
  title: null,
  models: [],
  selectedModel: null,
};

export function init({ models }) {
  state.models = models;
  state.selectedModel = models.find(m => m.default)?.name ?? models[0]?.name ?? null;
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

export function getSelectedModel() {
  return state.selectedModel;
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

  dropdown.innerHTML = `
    <div class="model-picker-dropdown-label">model</div>
    ${state.models.map(m => `
      <div class="model-picker-option${m.name === state.selectedModel ? ' selected' : ''}" data-name="${escapeHtml(m.name)}">
        <span class="model-picker-option-check">${m.name === state.selectedModel ? icon('check', 13) : ''}</span>
        <span class="model-picker-option-name">${escapeHtml(m.display_name)}</span>
      </div>
    `).join('')}
  `;

  dropdown.querySelectorAll('.model-picker-option').forEach(opt => {
    opt.addEventListener('click', () => {
      state.selectedModel = opt.dataset.name;
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
  return state.models.find(m => m.name === state.selectedModel)?.display_name ?? state.selectedModel ?? 'no model';
}

function escapeHtml(str) {
  return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
