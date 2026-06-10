import { admin } from './api.js';
import { requireAuth } from './settings-auth.js';
import { preload as preloadIcons, icon } from './icons.js';

let user = null;
let svgArrowLeft, svgUser, svgUsers, svgSliders;

async function init() {
  user = await requireAuth();
  if (!user) return;
  if (!user.is_admin) {
    window.location.href = '/settings/account';
    return;
  }
  await preloadIcons();
  svgArrowLeft = icon('arrow-left', 14);
  svgUser      = icon('user', 16);
  svgUsers     = icon('users', 16);
  svgSliders   = icon('sliders', 16);
  renderNav();
  renderPage();
}

// ── Nav ───────────────────────────────────────────────────────────────

function renderNav() {
  const path = window.location.pathname;

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
      ${icon('drama', 16)}
      Characters
    </a>
    <a href="/settings/notes" class="snav-item${path === '/settings/notes' ? ' active' : ''}">
      ${icon('file-text', 16)}
      Notes
    </a>
    <div class="snav-group-label">Admin</div>
    <a href="/settings/users" class="snav-item${path === '/settings/users' ? ' active' : ''}">
      ${svgUsers}
      Users
    </a>
    <a href="/settings/tools" class="snav-item${path === '/settings/tools' ? ' active' : ''}">
      ${svgSliders}
      Tools
    </a>
  `;
}

// ── Tools page ────────────────────────────────────────────────────────

function renderPage() {
  document.getElementById('smain').innerHTML = `
    <div class="smain-head">
      <h1>Tools</h1>
      <p>Database maintenance and diagnostics.</p>
    </div>
    <div class="section">
      <h2>Conversations</h2>
      <div class="tool-row">
        <div class="tool-row-info">
          <div class="tool-row-name">Import chat</div>
          <div class="tool-row-desc">Recreate a conversation from a JSON export as a new conversation on your account.</div>
        </div>
        <a href="/settings/import_chat" class="btn btn-secondary btn-sm">Import</a>
      </div>
    </div>
    <div class="section">
      <h2>Database</h2>
      <div class="tool-row" id="tool-orphaned-messages">
        <div class="tool-row-info">
          <div class="tool-row-name">Delete orphaned messages</div>
          <div class="tool-row-desc">Removes messages whose conversation has been deleted. Safe to run at any time.</div>
        </div>
        <button class="btn btn-secondary btn-sm" id="btn-orphaned-messages">Run</button>
      </div>
      <div id="orphaned-messages-result" class="tool-result hidden"></div>
    </div>
  `;

  document.getElementById('btn-orphaned-messages').addEventListener('click', runDeleteOrphanedMessages);
}

async function runDeleteOrphanedMessages() {
  if (!confirm('Delete all messages whose conversation has been deleted?\n\nThis cannot be undone.')) return;

  const btn    = document.getElementById('btn-orphaned-messages');
  const result = document.getElementById('orphaned-messages-result');

  btn.disabled    = true;
  btn.textContent = 'Running…';
  result.className = 'tool-result hidden';

  try {
    const data = await admin.tools.deleteOrphanedMessages();
    result.className   = 'tool-result tool-result--ok';
    result.textContent = data.deleted === 0
      ? 'No orphaned messages found.'
      : `Deleted ${data.deleted} orphaned message${data.deleted === 1 ? '' : 's'}.`;
  } catch (err) {
    result.className   = 'tool-result tool-result--error';
    result.textContent = 'Failed: ' + (err.message || 'unknown error');
  } finally {
    btn.disabled    = false;
    btn.textContent = 'Run';
    result.classList.remove('hidden');
  }
}

init();
