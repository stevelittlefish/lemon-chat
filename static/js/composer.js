import { icon } from './icons.js';

const composerContainer = document.getElementById('composer-container');

const ACCEPTED_IMAGE_TYPES = ['image/png', 'image/jpeg', 'image/gif', 'image/webp'];

let state = {
  onSend: null,
  onStop: null,
  onUpload: null,      // async (file) => attachment meta { id, filename, ... }
  streaming: false,
  userBlurred: false,
  canAttach: false,    // selected model supports image input
  attachments: [],     // uploaded-but-unsent attachment metas
  uploading: 0,        // in-flight upload count
};

let blurWatcher = null;

export function init({ onSend, onStop, onUpload }) {
  state.onSend = onSend;
  state.onStop = onStop;
  state.onUpload = onUpload;
  render();
}

// setCanAttach toggles the attach control based on the current model/character.
// Turning it off discards any pending (unsent) attachments.
export function setCanAttach(can) {
  state.canAttach = !!can;
  if (!can && state.attachments.length) {
    state.attachments = [];
    renderAttachments();
  }
  const attachBtn = composerContainer.querySelector('.composer-attach');
  if (attachBtn) attachBtn.hidden = !can;
  updateSendDisabled();
}

export function setStreaming(active) {
  state.streaming = active;
  const btn = composerContainer.querySelector('.composer-send');
  const textarea = composerContainer.querySelector('.composer-textarea');
  const attachBtn = composerContainer.querySelector('.composer-attach');
  if (btn) {
    if (active) {
      btn.innerHTML = icon('square');
      btn.disabled = false;
      btn.title = 'Stop';
    } else {
      btn.innerHTML = icon('send');
      btn.title = 'Send';
      updateSendDisabled();
    }
  }
  if (attachBtn) attachBtn.disabled = active;
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

function render() {
  composerContainer.innerHTML = `
    <div class="composer-attachments" id="composer-attachments" hidden></div>
    <div class="composer">
      <button class="composer-attach" id="composer-attach" title="Attach image" aria-label="Attach image" ${state.canAttach ? '' : 'hidden'}>
        ${icon('image')}
      </button>
      <textarea
        class="composer-textarea"
        id="composer-textarea"
        placeholder="send a message"
        rows="1"
      ></textarea>
      <button class="composer-send" id="composer-send" disabled title="Send">
        ${icon('send')}
      </button>
    </div>
    <input type="file" id="composer-file" accept="${ACCEPTED_IMAGE_TYPES.join(',')}" multiple hidden>
  `;

  const textarea = document.getElementById('composer-textarea');
  const sendBtn = document.getElementById('composer-send');
  const attachBtn = document.getElementById('composer-attach');
  const fileInput = document.getElementById('composer-file');

  textarea.addEventListener('input', () => {
    autoResize(textarea);
    updateSendDisabled();
  });

  textarea.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (!sendBtn.disabled) doSend();
    }
  });

  textarea.addEventListener('paste', (e) => {
    if (!state.canAttach) return;
    const files = imagesFromClipboard(e.clipboardData);
    if (files.length) {
      e.preventDefault();
      files.forEach(uploadOne);
    }
  });

  sendBtn.addEventListener('click', () => {
    if (state.streaming) state.onStop?.();
    else doSend();
  });

  attachBtn.addEventListener('click', () => fileInput.click());
  fileInput.addEventListener('change', () => {
    [...fileInput.files].forEach(uploadOne);
    fileInput.value = '';
  });
}

// imagesFromClipboard pulls pasted image files from a clipboard payload,
// covering both the .files list and the .items path (some sources — notably
// screenshots — expose the image only as a file-kind item with .files empty).
function imagesFromClipboard(data) {
  if (!data) return [];
  const seen = new Set();
  const out = [];
  const add = (file) => {
    if (!file || !ACCEPTED_IMAGE_TYPES.includes(file.type)) return;
    const key = `${file.name}:${file.size}:${file.type}`;
    if (seen.has(key)) return;
    seen.add(key);
    out.push(file);
  };
  for (const file of data.files || []) add(file);
  for (const item of data.items || []) {
    if (item.kind === 'file') add(item.getAsFile());
  }
  return out;
}

// uploadOne uploads a single file and, on success, adds a thumbnail.
async function uploadOne(file) {
  if (!state.onUpload || !ACCEPTED_IMAGE_TYPES.includes(file.type)) return;
  state.uploading++;
  const placeholder = { id: null, filename: file.name, uploading: true, url: URL.createObjectURL(file) };
  state.attachments.push(placeholder);
  renderAttachments();
  updateSendDisabled();
  try {
    const att = await state.onUpload(file);
    placeholder.id = att.id;
    placeholder.filename = att.filename;
    placeholder.uploading = false;
  } catch (err) {
    const i = state.attachments.indexOf(placeholder);
    if (i >= 0) state.attachments.splice(i, 1);
    console.error('attachment upload failed:', err);
  } finally {
    state.uploading--;
    renderAttachments();
    updateSendDisabled();
  }
}

function removeAttachment(att) {
  const i = state.attachments.indexOf(att);
  if (i >= 0) {
    if (att.url) URL.revokeObjectURL(att.url);
    state.attachments.splice(i, 1);
  }
  renderAttachments();
  updateSendDisabled();
}

function renderAttachments() {
  const strip = document.getElementById('composer-attachments');
  if (!strip) return;
  strip.hidden = state.attachments.length === 0;
  strip.innerHTML = '';
  for (const att of state.attachments) {
    const thumb = document.createElement('div');
    thumb.className = 'composer-thumb' + (att.uploading ? ' is-uploading' : '');

    const img = document.createElement('img');
    img.src = att.url || `/api/attachments/${att.id}`;
    img.alt = att.filename || 'attachment';
    thumb.appendChild(img);

    const remove = document.createElement('button');
    remove.type = 'button';
    remove.className = 'composer-thumb-remove';
    remove.title = 'Remove';
    remove.setAttribute('aria-label', 'Remove attachment');
    remove.innerHTML = icon('x', 12);
    remove.addEventListener('click', () => removeAttachment(att));
    thumb.appendChild(remove);

    strip.appendChild(thumb);
  }
}

// Sent attachments are the ready (uploaded) ones only.
function readyAttachments() {
  return state.attachments.filter(a => a.id != null && !a.uploading);
}

function updateSendDisabled() {
  const sendBtn = composerContainer.querySelector('.composer-send');
  const textarea = composerContainer.querySelector('.composer-textarea');
  if (!sendBtn || state.streaming) return;
  const hasText = (textarea?.value.trim() ?? '') !== '';
  const hasReady = readyAttachments().length > 0;
  sendBtn.disabled = (!hasText && !hasReady) || state.uploading > 0;
}

function doSend() {
  const textarea = document.getElementById('composer-textarea');
  const content = textarea.value.trim();
  const attachments = readyAttachments();
  if ((!content && attachments.length === 0) || state.streaming || state.uploading > 0) return;
  state.userBlurred = false;
  textarea.value = '';
  autoResize(textarea);
  textarea.focus();
  document.getElementById('composer-send').disabled = true;
  // Hand over a stable snapshot; clear the pending strip.
  state.attachments = [];
  renderAttachments();
  state.onSend?.(content, attachments);
}

function autoResize(el) {
  el.style.height = 'auto';
  el.style.height = Math.min(el.scrollHeight, 200) + 'px';
}
