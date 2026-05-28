import { render as renderMarkdown } from './markdown.js';
import { icon } from './icons.js';

const threadEl = document.getElementById('thread');
const container = document.getElementById('thread-container');

let userScrolledDuringStream = false;
let programmaticScroll = false;

function isNearBottom() {
  return container.scrollHeight - container.scrollTop - container.clientHeight < 40;
}

// Distinguish user-initiated scrolls from programmatic ones.
// scroll fires after position changes, so isNearBottom() is accurate here.
container.addEventListener('scroll', () => {
  if (programmaticScroll) return;
  if (!isNearBottom()) userScrolledDuringStream = true;
}, { passive: true });

export function showEmpty() {
  threadEl.innerHTML = `
    <div class="thread-empty">
      <img src="/assets/lemon-slice.svg" alt="">
      <p>start a conversation</p>
    </div>
  `;
}

export function renderMessages(msgs) {
  threadEl.innerHTML = '';
  if (!msgs.length) {
    showEmpty();
    return;
  }
  for (const msg of msgs) {
    threadEl.appendChild(buildMessage(msg));
  }
  scrollToBottom();
}

export function appendMessage(role, content, assistantName) {
  removeEmpty();
  const el = buildMessage({ role, content, name: assistantName });
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

  const roleEl = document.createElement('div');
  roleEl.className = 'message-role';
  roleEl.textContent = '';

  const contentEl = document.createElement('div');
  contentEl.className = 'message-content';
  contentEl.innerHTML = typingIndicatorHTML();

  wrapper.appendChild(roleEl);
  wrapper.appendChild(contentEl);
  threadEl.appendChild(wrapper);
  scrollToBottom();

  let accumulated = '';

  return {
    setName(name) {
      roleEl.textContent = name;
    },
    append(delta) {
      accumulated += delta;
      contentEl.innerHTML = renderMarkdown(accumulated) || typingIndicatorHTML();
      if (!userScrolledDuringStream) scrollToBottom();
    },
    finish() {
      userScrolledDuringStream = false;
      if (!accumulated) {
        wrapper.remove();
      } else {
        contentEl.innerHTML = renderMarkdown(accumulated);
      }
    },
    error(msg) {
      userScrolledDuringStream = false;
      contentEl.textContent = msg;
      contentEl.style.color = 'var(--danger)';
    },
  };
}

function buildMessage(msg) {
  const wrapper = document.createElement('div');
  wrapper.className = `message ${msg.role}`;

  if (msg.role !== 'user') {
    const roleEl = document.createElement('div');
    roleEl.className = 'message-role';
    roleEl.textContent = msg.name || msg.role;
    wrapper.appendChild(roleEl);
  }

  const contentEl = document.createElement('div');
  contentEl.className = 'message-content';
  contentEl.innerHTML = renderMarkdown(msg.content);
  wrapper.appendChild(contentEl);

  if (msg.role === 'assistant' && hasStats(msg)) {
    wrapper.appendChild(buildFooter(msg));
  }

  return wrapper;
}

function hasStats(msg) {
  return msg.prompt_tokens != null || msg.completion_tokens != null || msg.total_time_ms != null;
}

function buildFooter(msg) {
  const footer = document.createElement('div');
  footer.className = 'message-footer';

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
