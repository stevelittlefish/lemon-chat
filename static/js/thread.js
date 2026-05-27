import { render as renderMarkdown } from './markdown.js';

const threadEl = document.getElementById('thread');
const container = document.getElementById('thread-container');

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
    threadEl.appendChild(buildMessage(msg.role, msg.content));
  }
  scrollToBottom();
}

export function appendMessage(role, content) {
  removeEmpty();
  const el = buildMessage(role, content);
  threadEl.appendChild(el);
  scrollToBottom();
  return el;
}

// Adds a typing indicator and returns a controller to stream text into it.
export function startStreaming() {
  removeEmpty();
  const wrapper = document.createElement('div');
  wrapper.className = 'message assistant';

  const roleEl = document.createElement('div');
  roleEl.className = 'message-role';
  roleEl.textContent = 'assistant';

  const contentEl = document.createElement('div');
  contentEl.className = 'message-content';
  contentEl.innerHTML = typingIndicatorHTML();

  wrapper.appendChild(roleEl);
  wrapper.appendChild(contentEl);
  threadEl.appendChild(wrapper);
  scrollToBottom();

  let accumulated = '';

  return {
    append(delta) {
      accumulated += delta;
      contentEl.innerHTML = renderMarkdown(accumulated) || typingIndicatorHTML();
      scrollToBottom();
    },
    finish() {
      if (!accumulated) {
        wrapper.remove();
      } else {
        contentEl.innerHTML = renderMarkdown(accumulated);
      }
    },
    error(msg) {
      contentEl.textContent = msg;
      contentEl.style.color = 'var(--danger)';
    },
  };
}

function buildMessage(role, content) {
  const wrapper = document.createElement('div');
  wrapper.className = `message ${role}`;

  if (role !== 'user') {
    const roleEl = document.createElement('div');
    roleEl.className = 'message-role';
    roleEl.textContent = role;
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
  container.scrollTop = container.scrollHeight;
}

function typingIndicatorHTML() {
  return '<div class="typing-indicator"><span></span><span></span><span></span></div>';
}
