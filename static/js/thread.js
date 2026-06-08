import { render as renderMarkdown } from './markdown.js';
import { icon } from './icons.js';
import { messages as msgApi } from './api.js';
import { escapeHtml } from './utils.js';
import { createModal } from './modal.js';

// ── Artifact panel ───────────────────────────────────────────

const artifactPanel = document.getElementById('artifact-panel');
const artifactPanelTitle = document.getElementById('artifact-panel-title');
const artifactPanelFilename = document.getElementById('artifact-panel-filename');
const artifactPanelDownload = document.getElementById('artifact-panel-download');
const artifactPanelClose = document.getElementById('artifact-panel-close');
const artifactPanelBody = document.getElementById('artifact-panel-body');
const appEl = document.getElementById('app');

artifactPanelClose.addEventListener('click', closeArtifactPanel);

export function closeArtifactPanel() {
  artifactPanel.hidden = true;
  appEl.classList.remove('artifact-open');
}

async function openArtifactPanel(att) {
  artifactPanelTitle.textContent = att.title;
  artifactPanelFilename.textContent = att.filename;
  artifactPanelDownload.href = `/api/attachments/${att.id}?download=1`;
  artifactPanelDownload.setAttribute('download', att.filename);
  artifactPanelBody.className = 'artifact-panel-body';
  artifactPanelBody.innerHTML = `<p class="artifact-panel-loading">Loading…</p>`;
  artifactPanel.hidden = false;
  appEl.classList.add('artifact-open');

  try {
    const res = await fetch(`/api/attachments/${att.id}`);
    if (!res.ok) throw new Error(res.statusText);
    const text = await res.text();
    const ext = att.filename.split('.').pop().toLowerCase();
    if (ext === 'md') {
      artifactPanelBody.className = 'artifact-panel-body rendered-md';
      artifactPanelBody.innerHTML = renderMarkdown(text);
    } else {
      artifactPanelBody.className = 'artifact-panel-body';
      const pre = document.createElement('pre');
      pre.className = 'artifact-panel-pre';
      pre.textContent = text;
      artifactPanelBody.innerHTML = '';
      artifactPanelBody.appendChild(pre);
    }
  } catch {
    artifactPanelBody.innerHTML = `<p class="artifact-panel-loading">Could not load file.</p>`;
  }
}

function fileIconName(mimeType, filename) {
  const ext = (filename || '').split('.').pop().toLowerCase();
  const codeExts = ['py', 'js', 'ts', 'go', 'sh', 'html', 'css', 'json', 'rb', 'rs', 'cpp', 'c', 'java'];
  if (ext === 'md' || mimeType === 'text/markdown') return 'file-text';
  if (codeExts.includes(ext)) return 'file-code';
  return 'file';
}

function buildImagePlaceholder() {
  const div = document.createElement('div');
  div.className = 'image-placeholder';
  const label = document.createElement('span');
  label.className = 'image-placeholder-label';
  label.textContent = 'Generating image…';
  div.appendChild(label);
  return div;
}

let lightbox = null;
let lightboxClosing = false;
let lightboxImages = [];
let lightboxIndex = 0;

function isLightboxOpen() {
  return lightbox !== null && lightbox.classList.contains('is-open');
}

export function closeLightboxIfOpen() {
  if (lightboxClosing) {
    lightboxClosing = false;
    return true;
  }
  if (!isLightboxOpen()) return false;
  lightbox.classList.remove('is-open');
  return true;
}

function lightboxShow(index) {
  lightboxIndex = index;
  const { src, alt } = lightboxImages[index];
  lightbox.querySelector('.image-lightbox-img').src = src;
  lightbox.querySelector('.image-lightbox-img').alt = alt;
  lightbox.querySelector('.image-lightbox-prev').disabled = index === 0;
  lightbox.querySelector('.image-lightbox-next').disabled = index === lightboxImages.length - 1;
  const nav = lightbox.querySelector('.image-lightbox-nav');
  nav.classList.toggle('is-visible', lightboxImages.length >= 2);
  lightbox.querySelector('.image-lightbox-counter').textContent = `${index + 1} / ${lightboxImages.length}`;
}

function openLightbox(clickedSrc) {
  if (!lightbox) {
    lightbox = document.createElement('div');
    lightbox.className = 'image-lightbox';
    lightbox.setAttribute('role', 'dialog');
    lightbox.setAttribute('aria-modal', 'true');

    const closeBtn = document.createElement('button');
    closeBtn.className = 'image-lightbox-close';
    closeBtn.setAttribute('aria-label', 'Close');
    closeBtn.innerHTML = icon('x', 18);
    closeBtn.addEventListener('click', e => { e.stopPropagation(); closeLightbox(); });

    const body = document.createElement('div');
    body.className = 'image-lightbox-body';
    body.addEventListener('click', closeLightbox);

    const lbImg = document.createElement('img');
    lbImg.className = 'image-lightbox-img';
    lbImg.addEventListener('click', e => e.stopPropagation());

    body.appendChild(lbImg);

    const nav = document.createElement('div');
    nav.className = 'image-lightbox-nav';
    nav.addEventListener('click', e => e.stopPropagation());

    const inner = document.createElement('div');
    inner.className = 'image-lightbox-nav-inner';

    const prevBtn = document.createElement('button');
    prevBtn.className = 'image-lightbox-prev';
    prevBtn.setAttribute('aria-label', 'Previous image');
    prevBtn.innerHTML = icon('chevron-left', 22);
    prevBtn.addEventListener('click', () => lightboxShow(lightboxIndex - 1));

    const counter = document.createElement('span');
    counter.className = 'image-lightbox-counter';

    const nextBtn = document.createElement('button');
    nextBtn.className = 'image-lightbox-next';
    nextBtn.setAttribute('aria-label', 'Next image');
    nextBtn.innerHTML = icon('chevron-right', 22);
    nextBtn.addEventListener('click', () => lightboxShow(lightboxIndex + 1));

    inner.appendChild(prevBtn);
    inner.appendChild(counter);
    inner.appendChild(nextBtn);
    nav.appendChild(inner);

    lightbox.appendChild(closeBtn);
    lightbox.appendChild(body);
    lightbox.appendChild(nav);
    document.body.appendChild(lightbox);

    document.addEventListener('keydown', e => {
      if (!isLightboxOpen()) return;
      if (e.key === 'Escape') { closeLightbox(); return; }
      if (e.key === 'ArrowLeft'  && lightboxIndex > 0) lightboxShow(lightboxIndex - 1);
      if (e.key === 'ArrowRight' && lightboxIndex < lightboxImages.length - 1) lightboxShow(lightboxIndex + 1);
    });
  }

  lightboxImages = Array.from(document.querySelectorAll('.inline-image-img')).map(img => ({
    src: img.src,
    alt: img.alt,
  }));
  const idx = lightboxImages.findIndex(im => im.src === clickedSrc);
  lightboxShow(idx >= 0 ? idx : 0);
  lightbox.classList.add('is-open');
  history.pushState({ lightbox: true }, '');
}

function closeLightbox() {
  if (!isLightboxOpen()) return;
  lightbox.classList.remove('is-open');
  if (history.state?.lightbox) {
    lightboxClosing = true;
    history.back();
  }
}

function buildInlineImage(att) {
  const wrap = document.createElement('div');
  wrap.className = 'inline-image-wrap';

  const img = withRetry(document.createElement('img'));
  img.className = 'inline-image-img';
  img.src = `/api/attachments/${att.id}`;
  img.alt = att.title || 'Generated image';
  img.loading = 'lazy';
  img.addEventListener('click', () => openLightbox(img.src));

  const dlBtn = document.createElement('a');
  dlBtn.className = 'inline-image-download';
  dlBtn.href = `/api/attachments/${att.id}?download=1`;
  dlBtn.setAttribute('download', att.filename || 'image.png');
  dlBtn.setAttribute('aria-label', 'Download image');
  dlBtn.innerHTML = icon('download', 14);

  wrap.appendChild(img);
  wrap.appendChild(dlBtn);
  return wrap;
}

function buildAttachmentCard(att) {
  const card = document.createElement('div');
  card.className = 'attachment-card';

  const iconEl = document.createElement('div');
  iconEl.className = 'attachment-card-icon';
  iconEl.innerHTML = icon(fileIconName(att.mime_type, att.filename), 20);

  const info = document.createElement('div');
  info.className = 'attachment-card-info';
  const titleEl = document.createElement('div');
  titleEl.className = 'attachment-card-title';
  titleEl.textContent = att.title;
  const filenameEl = document.createElement('div');
  filenameEl.className = 'attachment-card-filename';
  filenameEl.textContent = att.filename;
  info.appendChild(titleEl);
  info.appendChild(filenameEl);

  const openBtn = document.createElement('button');
  openBtn.className = 'btn btn-secondary btn-sm attachment-card-open';
  openBtn.textContent = 'Open';
  openBtn.addEventListener('click', () => openArtifactPanel(att));

  card.appendChild(iconEl);
  card.appendChild(info);
  card.appendChild(openBtn);
  return card;
}

async function copyToClipboard(text) {
  if (navigator.clipboard) {
    await navigator.clipboard.writeText(text);
    return;
  }
  const ta = document.createElement('textarea');
  ta.value = text;
  ta.style.cssText = 'position:fixed;opacity:0;top:0;left:0';
  document.body.appendChild(ta);
  ta.focus();
  ta.select();
  document.execCommand('copy');
  ta.remove();
}

const threadEl = document.getElementById('thread');
const container = document.getElementById('thread-container');

let userScrolledDuringStream = false;
let programmaticScroll = false;
let forkHandler = null;
let currentConvId = null;

export function setConversationId(id) {
  currentConvId = id;
}

export function setForkHandler(fn) {
  forkHandler = fn;
}

// Avatar context — set by app.js whenever the active conversation changes.
let avatarCtx = {
  userId: null,
  userHasAvatar: false,
  characterId: null,       // for the streaming placeholder (no message yet)
  characterHasAvatar: false,
  characterMap: new Map(), // characterId → has_avatar, covers all loaded characters
};

export function setAvatarContext(ctx) {
  avatarCtx = { ...ctx };
}

// wheel fires before position changes and never fires for programmatic scrolls,
// so it reliably detects upward-scroll intent without the rAF race.
container.addEventListener('wheel', (e) => {
  if (e.deltaY < 0) userScrolledDuringStream = true;
}, { passive: true });

// Fallback for touch and keyboard (Page Up, arrow keys): scroll fires after
// position changes, so we guard against programmatic scrolls here.
container.addEventListener('scroll', () => {
  if (programmaticScroll) return;
  if (!isNearBottom()) userScrolledDuringStream = true;
}, { passive: true });

function isNearBottom() {
  return container.scrollHeight - container.scrollTop - container.clientHeight < 120;
}

export function showEmpty() {
  threadEl.innerHTML = `
    <div class="thread-empty">
      <img src="/assets/lemon-slice.svg" alt="">
      <p>start a conversation</p>
    </div>
  `;
}

export function showPicker(models, characters, onSelect) {
  const defaultChar = characters.find(c => c.is_default) ?? null;
  const restChars   = characters.filter(c => !c.is_default);

  function charCard(c) {
    return `
      <button class="picker-card${c.is_default ? ' picker-card--default' : ''}" data-type="character" data-id="${c.id}">
        ${c.is_default ? `<span class="picker-card-star">${icon('star', 11)}</span>` : ''}
        ${c.has_avatar
          ? `<div class="picker-card-avatar"><img src="/api/characters/${c.id}/avatar" alt=""></div>`
          : `<div class="picker-card-avatar picker-card-avatar--placeholder">${icon('drama', 20)}</div>`
        }
        <span class="picker-card-name">${escapeHtml(c.name)}</span>
      </button>`;
  }

  const modelCards = models.map(m => `
    <button class="picker-card" data-type="model" data-name="${escapeHtml(m.name)}">
      <span class="picker-card-icon">${icon('cpu', 22)}</span>
      <span class="picker-card-name">${escapeHtml(m.display_name)}</span>
    </button>
  `).join('');

  threadEl.innerHTML = `
    <div class="picker">
      ${defaultChar ? `
        <div class="picker-section">
          <div class="picker-section-label">default</div>
          <div class="picker-grid">${charCard(defaultChar)}</div>
        </div>
      ` : ''}
      <div class="picker-section">
        <div class="picker-section-label">models</div>
        <div class="picker-grid">${modelCards}</div>
      </div>
      ${restChars.length ? `
        <div class="picker-section">
          <div class="picker-section-label">characters</div>
          <div class="picker-grid">${restChars.map(charCard).join('')}</div>
        </div>
      ` : ''}
    </div>
  `;

  threadEl.querySelectorAll('.picker-card-avatar img').forEach(withRetry);

  threadEl.querySelectorAll('.picker-card').forEach(btn => {
    btn.addEventListener('click', () => {
      const sel = btn.dataset.type === 'model'
        ? { type: 'model', name: btn.dataset.name }
        : { type: 'character', id: Number(btn.dataset.id) };
      onSelect(sel);
    });
  });
}


export function renderMessages(msgs) {
  threadEl.innerHTML = '';
  // Skip role="tool" messages (merged into tool_interactions by the server).
  // Merge consecutive tool-call-only assistant messages into one bubble to match streaming.
  const visible = [];
  for (const msg of msgs) {
    if (msg.role === 'tool') continue;
    if (msg.role === 'assistant' && !msg.content && msg.tool_interactions?.length && visible.length > 0) {
      const last = visible[visible.length - 1];
      if (last.role === 'assistant') {
        visible[visible.length - 1] = {
          ...last,
          tool_interactions: [...(last.tool_interactions || []), ...msg.tool_interactions],
        };
        continue;
      }
    }
    visible.push(msg);
  }
  if (!visible.length) {
    showEmpty();
    return;
  }
  let lastDk = null;
  let prevSender = null;
  for (const msg of visible) {
    const dk = msgDateKey(msg.created_at);
    if (dk && dk !== lastDk) {
      threadEl.appendChild(buildDateSep(msg.created_at));
      lastDk = dk;
      prevSender = null;
    }
    const key = senderKey(msg.role, msg.character_id ?? null);
    threadEl.appendChild(buildMessage(msg, key === prevSender));
    prevSender = key;
  }
  scrollToBottom();
}

export function appendMessage(role, content, assistantName, characterId = null) {
  removeEmpty();
  const key = senderKey(role, characterId);
  const hideAvatar = key === lastThreadSenderKey();
  const el = buildMessage({ role, content, name: assistantName, character_id: characterId, created_at: new Date().toISOString() }, hideAvatar);
  threadEl.appendChild(el);
  scrollToBottom();
  return el;
}

// Adds a typing indicator and returns a controller to stream text into it.
export function startStreaming() {
  removeEmpty();
  userScrolledDuringStream = false;

  const wrapper = document.createElement('div');
  wrapper.className = 'message assistant';
  const streamSenderKey = senderKey('assistant', avatarCtx.characterId);
  wrapper.dataset.sender = streamSenderKey;

  const streamIsConsecutive = streamSenderKey === lastThreadSenderKey();
  const avatarEl = buildAvatar('assistant', avatarCtx.characterId, streamIsConsecutive);
  if (avatarEl) wrapper.appendChild(avatarEl);

  const colEl = document.createElement('div');
  colEl.className = 'message-col';

  const roleEl = streamIsConsecutive ? null : document.createElement('div');
  if (roleEl) {
    roleEl.className = 'message-role';
    roleEl.textContent = '';
    colEl.appendChild(roleEl);
  }

  const contentEl = document.createElement('div');
  contentEl.className = 'message-content';
  contentEl.innerHTML = typingIndicatorHTML();

  colEl.appendChild(contentEl);
  wrapper.appendChild(colEl);
  threadEl.appendChild(wrapper);
  scrollToBottom();

  let accumulated = '';
  let streamStats = null;
  let streamMsgId = null;
  const streamStartedAt = new Date().toISOString();
  let toolCallsEl = null; // lazy-created container for tool call entries
  const toolCallEls = new Map(); // id → .tool-call element for result updates
  const pendingImages = new Map(); // tool call id → placeholder element
  let hasImages = false; // true once any inline image is committed
  let waitingEl = null; // typing indicator shown while a tool call is running after text has streamed

  function showWaiting() {
    if (waitingEl) return;
    waitingEl = document.createElement('div');
    waitingEl.className = 'message-content';
    waitingEl.innerHTML = typingIndicatorHTML();
    colEl.appendChild(waitingEl);
  }

  function hideWaiting() {
    if (!waitingEl) return;
    waitingEl.remove();
    waitingEl = null;
  }

  return {
    setName(name) {
      if (roleEl) roleEl.textContent = name;
    },
    append(delta) {
      hideWaiting();
      accumulated += delta;
      contentEl.innerHTML = renderMarkdown(accumulated) || typingIndicatorHTML();
      if (!userScrolledDuringStream) scrollToBottom();
    },
    setStats(stats) {
      streamStats = stats;
    },
    setMessageId(id) {
      streamMsgId = id;
    },
    addToolCall(toolCall) {
      if (!toolCallsEl) {
        toolCallsEl = document.createElement('div');
        toolCallsEl.className = 'tool-calls';
        // If text has already streamed, place tool calls after the content element.
        if (accumulated) {
          colEl.appendChild(toolCallsEl);
        } else {
          colEl.insertBefore(toolCallsEl, contentEl);
        }
      }
      const el = buildToolCallEl(toolCall.name, toolCall.args ?? null, null);
      if (toolCall.id) toolCallEls.set(toolCall.id, el);
      toolCallsEl.appendChild(el);
      if (toolCall.name.startsWith('generate_image') && toolCall.id) {
        const placeholder = buildImagePlaceholder();
        if (accumulated) {
          colEl.insertBefore(placeholder, waitingEl ?? null);
        } else {
          colEl.insertBefore(placeholder, contentEl);
        }
        pendingImages.set(toolCall.id, placeholder);
      }
      if (accumulated) showWaiting();
      if (!userScrolledDuringStream) scrollToBottom();
    },
    updateToolCallResult({ id, result }) {
      const el = toolCallEls.get(id);
      if (!el) return;
      const details = el.querySelector('.tool-call-details');
      if (details) details.appendChild(buildToolCallSection('result', result));
    },
    addAttachment(att) {
      if (att.mime_type?.startsWith('image/') && att.tool_call_id) {
        const imgWrap = buildInlineImage(att);
        const imgTag = imgWrap.querySelector('img');
        if (imgTag) imgTag.addEventListener('load', () => { if (!userScrolledDuringStream) scrollToBottom(); }, { once: true });
        const placeholder = pendingImages.get(att.tool_call_id);
        if (placeholder) {
          placeholder.replaceWith(imgWrap);
          pendingImages.delete(att.tool_call_id);
        } else {
          // Fallback: no placeholder was pre-created — insert after tool calls.
          if (waitingEl) colEl.insertBefore(imgWrap, waitingEl);
          else if (accumulated) colEl.appendChild(imgWrap);
          else colEl.insertBefore(imgWrap, contentEl);
        }
        hasImages = true;
      } else {
        colEl.appendChild(buildAttachmentCard(att));
      }
      if (!userScrolledDuringStream) scrollToBottom();
    },
    finish() {
      hideWaiting();
      const shouldScroll = !userScrolledDuringStream;
      userScrolledDuringStream = false;
      if (!accumulated && !hasImages && !toolCallsEl) {
        wrapper.remove();
        return;
      }
      if (accumulated) {
        contentEl.innerHTML = renderMarkdown(accumulated);
        colEl.appendChild(buildFooter({ ...(streamStats || {}), role: 'assistant', id: streamMsgId, created_at: streamStartedAt }, accumulated));
      } else {
        // Tool calls / images only — remove the empty typing-indicator contentEl.
        contentEl.remove();
      }
      if (shouldScroll) scrollToBottom();
    },
    truncate() {
      userScrolledDuringStream = false;
      if (!accumulated && !hasImages) {
        wrapper.remove();
        return;
      }
      if (accumulated) {
        contentEl.innerHTML = renderMarkdown(accumulated);
        const stopNote = document.createElement('p');
        stopNote.className = 'message-stopped';
        stopNote.textContent = 'stopped';
        colEl.appendChild(stopNote);
        colEl.appendChild(buildFooter({ ...(streamStats || {}), role: 'assistant', id: streamMsgId, created_at: streamStartedAt }, accumulated));
      } else {
        contentEl.remove();
      }
    },
    error(msg) {
      userScrolledDuringStream = false;
      contentEl.textContent = msg;
      contentEl.style.color = 'var(--danger)';
    },
  };
}

function formatTime(isoStr) {
  if (!isoStr) return '';
  const d = new Date(isoStr);
  if (isNaN(d)) return '';
  const h = d.getHours();
  const m = String(d.getMinutes()).padStart(2, '0');
  return `${h % 12 || 12}:${m}${h < 12 ? 'am' : 'pm'}`;
}

function formatDate(isoStr) {
  if (!isoStr) return '';
  const d = new Date(isoStr);
  if (isNaN(d)) return '';
  const weekday = d.toLocaleDateString('en-GB', { weekday: 'long' });
  const day = d.getDate();
  const month = d.toLocaleDateString('en-GB', { month: 'long' });
  const label = `${weekday} ${day} ${month}`;
  return d.getFullYear() !== new Date().getFullYear() ? `${label} ${d.getFullYear()}` : label;
}

function msgDateKey(isoStr) {
  if (!isoStr) return '';
  const d = new Date(isoStr);
  if (isNaN(d)) return '';
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;
}

function buildDateSep(isoStr) {
  const el = document.createElement('div');
  el.className = 'date-sep';
  const span = document.createElement('span');
  span.textContent = formatDate(isoStr);
  el.appendChild(span);
  return el;
}

// Retry a failed image load once with a cache-busting param.
function withRetry(img) {
  img.onerror = () => {
    img.onerror = null;
    const base = img.src.split('?')[0];
    setTimeout(() => { img.src = `${base}?r=${Date.now()}`; }, 800);
  };
  return img;
}

function senderKey(role, characterId) {
  if (role === 'user') return 'user';
  return characterId ? `assistant:${characterId}` : 'assistant';
}

function lastThreadSenderKey() {
  const msgs = threadEl.querySelectorAll('.message[data-sender]');
  return msgs.length ? msgs[msgs.length - 1].dataset.sender : null;
}

function buildAvatar(role, msgCharacterId, ghost = false) {
  const el = document.createElement('div');
  el.className = ghost ? 'avatar-chat avatar-chat--ghost' : 'avatar-chat';
  if (ghost) return el;

  if (role === 'user') {
    if (avatarCtx.userHasAvatar && avatarCtx.userId) {
      const img = withRetry(document.createElement('img'));
      img.src = `/api/users/${avatarCtx.userId}/avatar`;
      img.alt = '';
      el.appendChild(img);
    } else {
      el.classList.add('avatar-chat--placeholder');
      el.innerHTML = icon('user', 28);
    }
    return el;
  }

  // assistant — use the character ID stored on the message (or passed explicitly for streaming)
  const charId = msgCharacterId;
  if (!charId) {
    el.classList.add('avatar-chat--placeholder');
    el.innerHTML = icon('cpu', 28);
    return el;
  }
  const hasAvatar = avatarCtx.characterMap.has(charId)
    ? avatarCtx.characterMap.get(charId)
    : (charId === avatarCtx.characterId ? avatarCtx.characterHasAvatar : false);
  if (hasAvatar) {
    const img = withRetry(document.createElement('img'));
    img.src = `/api/characters/${charId}/avatar`;
    img.alt = '';
    el.appendChild(img);
  } else {
    el.classList.add('avatar-chat--placeholder');
    el.innerHTML = icon('drama', 28);
  }
  return el;
}


function buildToolCallSection(label, text) {
  const section = document.createElement('div');
  section.className = 'tool-call-section';
  const labelEl = document.createElement('div');
  labelEl.className = 'tool-call-section-label';
  labelEl.textContent = label;
  const pre = document.createElement('pre');
  pre.className = 'tool-call-code';
  pre.textContent = text;
  section.appendChild(labelEl);
  section.appendChild(pre);
  return section;
}

function buildToolCallEl(name, args, result) {
  const container = document.createElement('div');
  container.className = 'tool-call';

  const toggle = document.createElement('button');
  toggle.className = 'tool-call-toggle';
  toggle.setAttribute('aria-expanded', 'false');

  const iconEl = document.createElement('span');
  iconEl.className = 'tool-call-icon';
  iconEl.innerHTML = icon('settings', 12);

  const nameEl = document.createElement('span');
  nameEl.className = 'tool-call-name';
  nameEl.textContent = name;

  const chevron = document.createElement('span');
  chevron.className = 'tool-call-chevron';
  chevron.innerHTML = icon('chevron-down', 12);

  toggle.appendChild(iconEl);
  toggle.appendChild(nameEl);
  toggle.appendChild(chevron);

  const details = document.createElement('div');
  details.className = 'tool-call-details';
  details.hidden = true;

  if (args !== null && args !== undefined) {
    const argsText = typeof args === 'string' ? args : JSON.stringify(args, null, 2);
    details.appendChild(buildToolCallSection('args', argsText));
  }

  if (result !== null && result !== undefined) {
    details.appendChild(buildToolCallSection('result', result));
  }

  toggle.addEventListener('click', () => {
    const expanded = toggle.getAttribute('aria-expanded') === 'true';
    toggle.setAttribute('aria-expanded', String(!expanded));
    details.hidden = expanded;
  });

  container.appendChild(toggle);
  container.appendChild(details);
  return container;
}

function buildMessage(msg, hideAvatar = false) {
  const wrapper = document.createElement('div');
  wrapper.className = `message ${msg.role}`;
  wrapper.dataset.sender = senderKey(msg.role, msg.character_id ?? null);

  const avatarEl = buildAvatar(msg.role, msg.character_id ?? null, hideAvatar);
  if (avatarEl) wrapper.appendChild(avatarEl);

  const colEl = document.createElement('div');
  colEl.className = 'message-col';

  if (!hideAvatar) {
    const roleEl = document.createElement('div');
    roleEl.className = 'message-role';
    roleEl.textContent = msg.role === 'user' ? (msg.name || 'you') : (msg.name || msg.role);
    colEl.appendChild(roleEl);
  }

  if (msg.content) {
    const contentEl = document.createElement('div');
    contentEl.className = 'message-content';
    contentEl.innerHTML = renderMarkdown(msg.content);
    colEl.appendChild(contentEl);
  }

  if (msg.tool_interactions?.length) {
    const tcEl = document.createElement('div');
    tcEl.className = 'tool-calls';
    for (const ti of msg.tool_interactions) {
      const args = ti.args ?? null;
      const argsText = args != null ? JSON.stringify(args, null, 2) : null;
      tcEl.appendChild(buildToolCallEl(ti.name, argsText, ti.result || null));
    }
    colEl.appendChild(tcEl);
    for (const ti of msg.tool_interactions) {
      if (!ti.attachment) continue;
      if (ti.attachment.mime_type?.startsWith('image/')) {
        colEl.appendChild(buildInlineImage(ti.attachment));
      } else {
        colEl.appendChild(buildAttachmentCard(ti.attachment));
      }
    }
  }

  colEl.appendChild(buildFooter(msg, msg.content));

  wrapper.appendChild(colEl);
  return wrapper;
}

function hasStats(msg) {
  return msg.prompt_tokens != null || msg.completion_tokens != null || msg.total_time_ms != null;
}

// ── Markdown modal ───────────────────────────────────────────

let _mdModal = null;

function getMdModal() {
  if (_mdModal) return _mdModal;

  const { overlay, dialog, open, close } = createModal('Raw markdown');
  overlay.className = 'md-modal';
  dialog.className = 'md-modal-dialog';

  const head = document.createElement('div');
  head.className = 'md-modal-head';

  const eyebrow = document.createElement('span');
  eyebrow.className = 'md-modal-eyebrow';
  eyebrow.textContent = 'Raw markdown';

  const actions = document.createElement('div');
  actions.className = 'md-modal-actions';

  const pre = document.createElement('pre');
  pre.className = 'md-modal-pre';

  const copyBtn = document.createElement('button');
  copyBtn.className = 'btn btn-secondary btn-sm';
  const copyLabel = document.createElement('span');
  copyLabel.textContent = 'Copy';
  copyBtn.appendChild(copyLabel);
  copyBtn.addEventListener('click', async () => {
    await copyToClipboard(pre.textContent);
    copyLabel.textContent = 'Copied';
    setTimeout(() => { copyLabel.textContent = 'Copy'; }, 1500);
  });

  const closeBtn = document.createElement('button');
  closeBtn.className = 'md-modal-x';
  closeBtn.setAttribute('aria-label', 'Close');
  closeBtn.innerHTML = icon('x', 14);
  closeBtn.addEventListener('click', close);

  actions.appendChild(copyBtn);
  actions.appendChild(closeBtn);
  head.appendChild(eyebrow);
  head.appendChild(actions);

  dialog.appendChild(head);
  dialog.appendChild(pre);

  _mdModal = { overlay, open, close, pre };
  return _mdModal;
}

function openMdModal(content) {
  const modal = getMdModal();
  modal.pre.textContent = content;
  modal.open();
}

// ── Fork confirmation modal ──────────────────────────────────

let _forkModal = null;

function getForkModal() {
  if (_forkModal) return _forkModal;

  const { overlay, dialog, open, close } = createModal('Duplicate conversation');
  overlay.className = 'fork-modal';
  dialog.className = 'fork-modal-dialog';

  const title = document.createElement('p');
  title.className = 'fork-modal-title';
  title.textContent = 'Duplicate conversation';

  const body = document.createElement('p');
  body.className = 'fork-modal-body';
  body.textContent = 'This will create a new conversation containing all messages up to and including this one. The title will be prefixed with "copy:".';

  const actions = document.createElement('div');
  actions.className = 'fork-modal-actions';

  const cancelBtn = document.createElement('button');
  cancelBtn.className = 'btn btn-secondary btn-sm';
  cancelBtn.textContent = 'Cancel';
  cancelBtn.addEventListener('click', close);

  const confirmBtn = document.createElement('button');
  confirmBtn.className = 'btn btn-primary btn-sm';
  confirmBtn.textContent = 'Duplicate';

  actions.appendChild(cancelBtn);
  actions.appendChild(confirmBtn);
  dialog.appendChild(title);
  dialog.appendChild(body);
  dialog.appendChild(actions);

  _forkModal = { overlay, open, close, confirmBtn };
  return _forkModal;
}

function showForkConfirm(messageId) {
  const modal = getForkModal();
  modal.confirmBtn.disabled = false;
  modal.confirmBtn.textContent = 'Duplicate';
  modal.confirmBtn.onclick = async () => {
    modal.confirmBtn.disabled = true;
    modal.confirmBtn.textContent = 'Duplicating…';
    try {
      await forkHandler?.(messageId);
    } finally {
      modal.close();
    }
  };
  modal.open();
}

// ── Context modal ────────────────────────────────────────────

let _ctxModal = null;

function getCtxModal() {
  if (_ctxModal) return _ctxModal;

  const { overlay, dialog, open, close } = createModal('LLM context');
  overlay.className = 'ctx-modal';
  dialog.className = 'ctx-modal-dialog';

  const head = document.createElement('div');
  head.className = 'md-modal-head';

  const eyebrow = document.createElement('span');
  eyebrow.className = 'md-modal-eyebrow';
  eyebrow.textContent = 'LLM context';

  const actions = document.createElement('div');
  actions.className = 'md-modal-actions';

  const copyBtn = document.createElement('button');
  copyBtn.className = 'btn btn-secondary btn-sm';
  const copyLabel = document.createElement('span');
  copyLabel.textContent = 'Copy JSON';
  copyBtn.appendChild(copyLabel);
  copyBtn.addEventListener('click', async () => {
    if (!_ctxModal?._messages) return;
    await copyToClipboard(JSON.stringify(_ctxModal._messages, null, 2));
    copyLabel.textContent = 'Copied';
    setTimeout(() => { copyLabel.textContent = 'Copy JSON'; }, 1500);
  });

  const closeBtn = document.createElement('button');
  closeBtn.className = 'md-modal-x';
  closeBtn.setAttribute('aria-label', 'Close');
  closeBtn.innerHTML = icon('x', 14);
  closeBtn.addEventListener('click', close);

  actions.appendChild(copyBtn);
  actions.appendChild(closeBtn);
  head.appendChild(eyebrow);
  head.appendChild(actions);

  const body = document.createElement('div');
  body.className = 'ctx-modal-body';

  dialog.appendChild(head);
  dialog.appendChild(body);

  _ctxModal = { overlay, open, close, body, _messages: null };
  return _ctxModal;
}

async function openCtxModal(convId, msgId) {
  const modal = getCtxModal();
  modal.body.innerHTML = '<p class="ctx-modal-loading">Loading…</p>';
  modal.open();
  try {
    const { messages } = await msgApi.context(convId, msgId);
    modal._messages = messages;
    modal.body.innerHTML = '';
    for (const msg of messages) {
      const block = document.createElement('div');
      block.className = 'ctx-msg-block';

      const roleEl = document.createElement('div');
      roleEl.className = 'ctx-msg-role';
      let roleLabel = msg.role;
      if (msg.tool_call_id) roleLabel += ` (id: ${msg.tool_call_id})`;
      roleEl.textContent = roleLabel;
      block.appendChild(roleEl);

      if (msg.content) {
        const pre = document.createElement('pre');
        pre.className = 'ctx-msg-content';
        pre.textContent = msg.content;
        block.appendChild(pre);
      }
      if (msg.tool_calls) {
        const pre = document.createElement('pre');
        pre.className = 'ctx-msg-content';
        pre.textContent = JSON.stringify(msg.tool_calls, null, 2);
        block.appendChild(pre);
      }
      if (!msg.content && !msg.tool_calls) {
        const pre = document.createElement('pre');
        pre.className = 'ctx-msg-content';
        pre.textContent = '';
        block.appendChild(pre);
      }

      modal.body.appendChild(block);
    }
  } catch (err) {
    modal.body.innerHTML = `<p class="ctx-modal-loading">Failed to load context: ${escapeHtml(err.message)}</p>`;
  }
}

// ── Message footer ───────────────────────────────────────────

function buildFooter(msg, content) {
  const footer = document.createElement('div');
  footer.className = 'message-footer';

  if (msg.created_at) {
    const tsEl = document.createElement('div');
    tsEl.className = 'message-ts';
    tsEl.textContent = formatTime(msg.created_at);
    footer.appendChild(tsEl);
  }

  const clipBtn = document.createElement('button');
  clipBtn.className = 'foot-btn';
  clipBtn.title = 'View raw markdown';
  clipBtn.setAttribute('aria-label', 'View raw markdown');
  clipBtn.innerHTML = icon('code');
  clipBtn.addEventListener('click', () => openMdModal(content));
  footer.appendChild(clipBtn);

  if (msg.role === 'assistant' && msg.id) {
    const forkBtn = document.createElement('button');
    forkBtn.className = 'foot-btn';
    forkBtn.title = 'Duplicate conversation to this point';
    forkBtn.setAttribute('aria-label', 'Duplicate conversation to this point');
    forkBtn.innerHTML = icon('fork', 14);
    forkBtn.addEventListener('click', () => showForkConfirm(msg.id));
    footer.appendChild(forkBtn);
  }

  const convId = msg.conversation_id ?? currentConvId;
  if (msg.role === 'assistant' && msg.id && convId) {
    const ctxBtn = document.createElement('button');
    ctxBtn.className = 'foot-btn';
    ctxBtn.title = 'View LLM context';
    ctxBtn.setAttribute('aria-label', 'View LLM context');
    ctxBtn.innerHTML = icon('list', 14);
    ctxBtn.addEventListener('click', () => openCtxModal(convId, msg.id));
    footer.appendChild(ctxBtn);
  }

  if (hasStats(msg)) {
    const infoWrap = document.createElement('div');
    infoWrap.className = 'info-wrap';

    const btn = document.createElement('button');
    btn.className = 'foot-btn';
    btn.title = 'Token usage';
    btn.setAttribute('aria-label', 'Token usage');
    btn.innerHTML = icon('info', 14);

    btn.addEventListener('click', () => {
      const existing = infoWrap.querySelector('.info-pop');
      if (existing) {
        existing.remove();
        btn.classList.remove('active');
      } else {
        btn.classList.add('active');
        const openUp = btn.closest('.message') === threadEl.lastElementChild;
        const pop = buildInfoPop(msg, () => {
          pop.remove();
          btn.classList.remove('active');
        }, openUp);
        infoWrap.appendChild(pop);
      }
    });

    infoWrap.appendChild(btn);
    footer.appendChild(infoWrap);
  }

  return footer;
}

function buildInfoPop(msg, onClose, openUp = false) {
  const pop = document.createElement('div');
  pop.className = openUp ? 'info-pop up' : 'info-pop';
  pop.setAttribute('role', 'dialog');
  pop.setAttribute('aria-label', 'Token usage');

  const fmtMs = (ms) => ms >= 1000 ? (ms / 1000).toFixed(2) + ' s' : ms + ' ms';
  const tokensPerSec = (msg.total_time_ms && msg.completion_tokens)
    ? Math.round(msg.completion_tokens / (msg.total_time_ms / 1000))
    : null;

  const head = document.createElement('div');
  head.className = 'info-pop-head';
  const eyebrow = document.createElement('span');
  eyebrow.className = 'info-pop-eyebrow';
  eyebrow.textContent = 'Token usage';
  const closeBtn = document.createElement('button');
  closeBtn.className = 'info-pop-x';
  closeBtn.setAttribute('aria-label', 'Close');
  closeBtn.innerHTML = icon('x', 12);
  closeBtn.addEventListener('click', onClose);
  head.appendChild(eyebrow);
  head.appendChild(closeBtn);

  const stats = document.createElement('dl');
  stats.className = 'info-pop-stats';

  const addStat = (label, value, unit) => {
    const div = document.createElement('div');
    const dt = document.createElement('dt');
    dt.textContent = label;
    const dd = document.createElement('dd');
    dd.textContent = value;
    if (unit) {
      const span = document.createElement('span');
      span.textContent = unit;
      dd.appendChild(span);
    }
    div.appendChild(dt);
    div.appendChild(dd);
    stats.appendChild(div);
  };

  if (msg.prompt_tokens != null) addStat('Prompt', msg.prompt_tokens.toLocaleString(), 'tok');
  if (msg.completion_tokens != null) addStat('Response', msg.completion_tokens.toLocaleString(), 'tok');
  if (msg.prompt_tokens != null && msg.completion_tokens != null) addStat('Total', (msg.prompt_tokens + msg.completion_tokens).toLocaleString(), 'tok');
  if (msg.total_time_ms != null) addStat('Total time', fmtMs(msg.total_time_ms));
  if (tokensPerSec != null) addStat('Throughput', tokensPerSec, 'tok/s');

  pop.appendChild(head);
  pop.appendChild(stats);

  // Close on outside click (deferred so the opening click doesn't immediately close it)
  setTimeout(() => {
    const onDoc = (e) => {
      if (!pop.isConnected) { document.removeEventListener('mousedown', onDoc); return; }
      if (!pop.contains(e.target)) { onClose(); document.removeEventListener('mousedown', onDoc); }
    };
    document.addEventListener('mousedown', onDoc);
  }, 0);

  return pop;
}

function removeEmpty() {
  const empty = threadEl.querySelector('.thread-empty');
  if (empty) empty.remove();
}

function scrollToBottom() {
  programmaticScroll = true;
  container.scrollTop = container.scrollHeight;
  // Reset after the scroll event fires (next animation frame is sufficient).
  requestAnimationFrame(() => { programmaticScroll = false; });
}

function typingIndicatorHTML() {
  return '<div class="typing-indicator"><span></span><span></span><span></span></div>';
}
