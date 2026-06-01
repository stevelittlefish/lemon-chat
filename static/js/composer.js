import { icon } from './icons.js';

const composerContainer = document.getElementById('composer-container');

let state = {
  onSend: null,
  onStop: null,
  streaming: false,
  userBlurred: false,
};

let blurWatcher = null;

export function init({ onSend, onStop }) {
  state.onSend = onSend;
  state.onStop = onStop;
  render();
}

export function setStreaming(active) {
  state.streaming = active;
  const btn = composerContainer.querySelector('.composer-send');
  const textarea = composerContainer.querySelector('.composer-textarea');
  if (btn) {
    if (active) {
      btn.innerHTML = icon('square');
      btn.disabled = false;
      btn.title = 'Stop';
    } else {
      btn.innerHTML = icon('send');
      btn.disabled = textarea?.value.trim() === '';
      btn.title = 'Send';
    }
  }
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
    <div class="composer">
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

  sendBtn.addEventListener('click', () => {
    if (state.streaming) state.onStop?.();
    else doSend();
  });
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
  state.onSend?.(content);
}

function autoResize(el) {
  el.style.height = 'auto';
  el.style.height = Math.min(el.scrollHeight, 200) + 'px';
}
