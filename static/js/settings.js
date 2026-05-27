import { auth } from './api.js';

let user = null;

// ── Init ──────────────────────────────────────────────────────────────

async function init() {
  try {
    user = await auth.me();
  } catch {
    window.location.href = '/';
    return;
  }
  renderNav();
  showAccountPanel();
}

// ── Nav ───────────────────────────────────────────────────────────────

function renderNav() {
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
    <div class="snav-item active">
      ${svgUser}
      Account
    </div>
  `;
}

// ── Account panel ─────────────────────────────────────────────────────

function showAccountPanel() {
  document.getElementById('smain').innerHTML = `
    <div class="smain-head">
      <h1>Account</h1>
      <p>Your profile and the keys to the kingdom.</p>
    </div>
    <div class="section">
      <h2>Profile</h2>
      <p class="lead">Your account on this server.</p>
      <div class="field">
        <div class="meta">
          <span class="lbl">Username</span>
          <span class="desc">Contact an admin to change it.</span>
        </div>
        <div class="ctl">
          <span class="mono-tag">${escapeHtml(user.username)}</span>
        </div>
      </div>
    </div>
    <div class="section">
      <h2>Password</h2>
      <p class="lead">Protects your account when signing in.</p>
      <div id="pwd-section"></div>
    </div>
  `;
  showPwdDefault();
}

// ── Password section ──────────────────────────────────────────────────

function showPwdDefault(successMsg) {
  const section = document.getElementById('pwd-section');
  section.innerHTML = `
    ${successMsg ? `<div class="field-msg field-msg--success">${escapeHtml(successMsg)}</div>` : ''}
    <div class="field">
      <div class="meta">
        <span class="lbl">Password</span>
        <span class="desc">${user.has_password ? 'A password is set.' : 'No password set.'}</span>
      </div>
      <div class="ctl">
        <button class="btn btn-secondary btn-sm" id="open-pwd-btn">
          ${svgLock}
          ${user.has_password ? 'Change password' : 'Set password'}
        </button>
      </div>
    </div>
  `;
  document.getElementById('open-pwd-btn').addEventListener('click', showPwdForm);
}

function showPwdForm() {
  const section = document.getElementById('pwd-section');
  section.innerHTML = `
    <div class="pwd-form">
      ${user.has_password ? `
        <div class="pwd-row">
          <label for="pwd-current">Current password</label>
          <div class="pwd-input-wrap">
            <input id="pwd-current" type="password" class="input" placeholder="••••••••" autocomplete="current-password">
          </div>
        </div>
      ` : ''}
      <div class="pwd-row">
        <label for="pwd-new">New password</label>
        <div class="pwd-input-wrap">
          <input id="pwd-new" type="password" class="input" placeholder="at least 6 characters" autocomplete="new-password">
          <button class="pwd-toggle" id="pwd-toggle" type="button" title="Show password">
            ${svgEye}
          </button>
        </div>
      </div>
      <div class="pwd-row">
        <label for="pwd-confirm">Confirm password</label>
        <div class="pwd-input-wrap">
          <input id="pwd-confirm" type="password" class="input" placeholder="type it again" autocomplete="new-password">
        </div>
      </div>
      <div class="pwd-strength" id="pwd-strength" hidden>
        <span class="strength-bar"><span class="strength-fill" id="strength-fill"></span></span>
        <span class="strength-label" id="strength-label"></span>
      </div>
      <div class="field-msg field-msg--error" id="pwd-error" hidden></div>
      <div class="pwd-actions">
        <button class="btn btn-ghost btn-sm" id="cancel-pwd-btn" type="button">Cancel</button>
        <button class="btn btn-primary btn-sm" id="save-pwd-btn" type="button">Update password</button>
      </div>
    </div>
  `;

  const newInput     = document.getElementById('pwd-new');
  const strengthEl   = document.getElementById('pwd-strength');
  const fillEl       = document.getElementById('strength-fill');
  const labelEl      = document.getElementById('strength-label');
  const toggleBtn    = document.getElementById('pwd-toggle');
  const errorEl      = document.getElementById('pwd-error');
  const cancelBtn    = document.getElementById('cancel-pwd-btn');
  const saveBtn      = document.getElementById('save-pwd-btn');

  newInput.addEventListener('input', () => {
    if (!newInput.value) {
      strengthEl.hidden = true;
      return;
    }
    const { level, label } = passwordStrength(newInput.value);
    strengthEl.hidden = false;
    fillEl.className = `strength-fill ${level}`;
    labelEl.textContent = label;
  });

  let showPwd = false;
  toggleBtn.addEventListener('click', () => {
    showPwd = !showPwd;
    const type = showPwd ? 'text' : 'password';
    newInput.type = type;
    const confirmInput = document.getElementById('pwd-confirm');
    if (confirmInput) confirmInput.type = type;
  });

  cancelBtn.addEventListener('click', () => showPwdDefault());

  saveBtn.addEventListener('click', async () => {
    errorEl.hidden = true;

    const currentInput = document.getElementById('pwd-current');
    const confirmInput = document.getElementById('pwd-confirm');
    const currentPwd   = currentInput?.value ?? '';
    const newPwd       = newInput.value;
    const confirmPwd   = confirmInput?.value ?? '';

    if (newPwd.length < 6) {
      showError('New password must be at least 6 characters.');
      return;
    }
    if (newPwd !== confirmPwd) {
      showError('Passwords do not match.');
      return;
    }

    saveBtn.disabled = true;
    saveBtn.textContent = 'Saving…';

    try {
      await auth.changePassword(currentPwd, newPwd);
      user = { ...user, has_password: true };
      showPwdDefault('Password updated.');
    } catch (err) {
      saveBtn.disabled = false;
      saveBtn.textContent = 'Update password';
      showError(err.message);
    }
  });

  function showError(msg) {
    errorEl.hidden = false;
    errorEl.textContent = msg;
  }
}

// ── Helpers ───────────────────────────────────────────────────────────

function passwordStrength(pwd) {
  if (pwd.length < 3) return { level: 'weak', label: 'Too short.' };
  let score = 0;
  if (pwd.length >= 12) score += 2;
  else if (pwd.length >= 6) score++;
  if (/[A-Z]/.test(pwd)) score++;
  if (/[0-9]/.test(pwd)) score++;
  if (/[^A-Za-z0-9]/.test(pwd)) score++;
  if (score <= 2) return { level: 'weak',   label: 'Weak. Try a longer passphrase.' };
  if (score <= 3) return { level: 'medium', label: 'Decent. A long passphrase is better than a clever short one.' };
  return             { level: 'strong', label: 'Strong.' };
}

function escapeHtml(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

// ── Icons (inline Lucide, 1.8px stroke) ──────────────────────────────

const svgArrowLeft = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M19 12H5M12 19l-7-7 7-7"/></svg>`;

const svgUser = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>`;

const svgLock = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V7a4 4 0 0 1 8 0v4"/></svg>`;

const svgEye = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>`;

init();
