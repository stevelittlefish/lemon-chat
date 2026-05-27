const composerContainer = document.getElementById('composer-container');

let state = {
  models: [],
  selectedModel: null,
  onSend: null,
  streaming: false,
  userBlurred: false,
};

let blurWatcher = null;

export function init({ models, onSend }) {
  state.models = models;
  state.selectedModel = models.find(m => m.default)?.name ?? models[0]?.name ?? null;
  state.onSend = onSend;
  render();
}

export function setStreaming(active) {
  state.streaming = active;
  const btn = composerContainer.querySelector('.composer-send');
  const textarea = composerContainer.querySelector('.composer-textarea');
  if (btn) btn.disabled = active;
  if (textarea) {
    textarea.disabled = active;
    if (active) {
      blurWatcher = (e) => {
        if (!composerContainer.contains(e.target)) state.userBlurred = true;
      };
      document.addEventListener('pointerdown', blurWatcher);
    } else {
      if (blurWatcher) {
        document.removeEventListener('pointerdown', blurWatcher);
        blurWatcher = null;
      }
      if (!state.userBlurred) textarea.focus();
      state.userBlurred = false;
    }
  }
}

export function getSelectedModel() {
  return state.selectedModel;
}

function render() {
  composerContainer.innerHTML = `
    <div class="composer-meta">
      <button class="model-picker" id="model-picker">
        ${iconCpu()}
        <span id="model-label">${escapeHtml(selectedDisplayName())}</span>
        ${iconChevronDown()}
      </button>
    </div>
    <div class="composer">
      <textarea
        class="composer-textarea"
        id="composer-textarea"
        placeholder="send a message"
        rows="1"
      ></textarea>
      <button class="composer-send" id="composer-send" disabled title="Send">
        ${iconSend()}
      </button>
    </div>
  `;

  const textarea = document.getElementById('composer-textarea');
  const sendBtn = document.getElementById('composer-send');

  textarea.addEventListener('input', () => {
    autoResize(textarea);
    sendBtn.disabled = textarea.value.trim() === '' || state.streaming;
  });

  textarea.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (!sendBtn.disabled) doSend();
    }
  });

  sendBtn.addEventListener('click', doSend);

  document.getElementById('model-picker').addEventListener('click', openModelPicker);
}

function doSend() {
  const textarea = document.getElementById('composer-textarea');
  const content = textarea.value.trim();
  if (!content || state.streaming) return;
  state.userBlurred = false;
  textarea.value = '';
  autoResize(textarea);
  textarea.focus();
  document.getElementById('composer-send').disabled = true;
  state.onSend?.(content, state.selectedModel);
}

function openModelPicker() {
  // Remove existing picker if open
  const existing = document.getElementById('model-dropdown');
  if (existing) { existing.remove(); return; }

  const btn = document.getElementById('model-picker');
  const rect = btn.getBoundingClientRect();

  const dropdown = document.createElement('div');
  dropdown.id = 'model-dropdown';
  dropdown.className = 'menu';
  dropdown.style.cssText = `
    position: fixed;
    left: ${rect.left}px;
    bottom: ${window.innerHeight - rect.top + 4}px;
    min-width: 180px;
    z-index: 100;
  `;

  dropdown.innerHTML = state.models.map(m => `
    <button class="menu-item${m.name === state.selectedModel ? ' active' : ''}" data-name="${escapeHtml(m.name)}">
      ${escapeHtml(m.display_name)}
    </button>
  `).join('');

  dropdown.querySelectorAll('.menu-item').forEach(item => {
    item.addEventListener('click', () => {
      state.selectedModel = item.dataset.name;
      document.getElementById('model-label').textContent = selectedDisplayName();
      dropdown.remove();
    });
  });

  document.body.appendChild(dropdown);
  setTimeout(() => document.addEventListener('click', () => dropdown.remove(), { once: true }), 0);
}

function autoResize(el) {
  el.style.height = 'auto';
  el.style.height = Math.min(el.scrollHeight, 200) + 'px';
}

function selectedDisplayName() {
  return state.models.find(m => m.name === state.selectedModel)?.display_name ?? state.selectedModel ?? 'no model';
}

function escapeHtml(str) {
  return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function iconSend() {
  return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>`;
}

function iconCpu() {
  return `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="4" width="16" height="16" rx="2"/><rect x="9" y="9" width="6" height="6"/><line x1="9" y1="1" x2="9" y2="4"/><line x1="15" y1="1" x2="15" y2="4"/><line x1="9" y1="20" x2="9" y2="23"/><line x1="15" y1="20" x2="15" y2="23"/><line x1="20" y1="9" x2="23" y2="9"/><line x1="20" y1="14" x2="23" y2="14"/><line x1="1" y1="9" x2="4" y2="9"/><line x1="1" y1="14" x2="4" y2="14"/></svg>`;
}

function iconChevronDown() {
  return `<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>`;
}
