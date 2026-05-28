import { render as renderMarkdown } from './markdown.js';

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
    threadEl.appendChild(buildMessage(msg.role, msg.content, msg.name));
  }
  scrollToBottom();
}

export function appendMessage(role, content, assistantName) {
  removeEmpty();
  const el = buildMessage(role, content, assistantName);
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

function buildMessage(role, content, assistantName) {
  const wrapper = document.createElement('div');
  wrapper.className = `message ${role}`;

  if (role !== 'user') {
    const roleEl = document.createElement('div');
    roleEl.className = 'message-role';
    roleEl.textContent = assistantName || role;
    wrapper.appendChild(roleEl);
  }

  const contentEl = document.createElement('div');
  contentEl.className = 'message-content';
  contentEl.innerHTML = renderMarkdown(content);
  wrapper.appendChild(contentEl);
  return wrapper;
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
