import { auth } from './api.js';
import { preload as preloadIcons, icon } from './icons.js';
import { renderAvatarSection, attachAvatarSection } from './settings-avatar.js';

let user = null;
let svgArrowLeft, svgUser, svgUsers, svgCpu, svgLock, svgEye, svgPencil;

async function init() {
  try {
    user = await auth.me();
  } catch {
    window.location.href = '/';
    return;
  }
  await preloadIcons();
  svgArrowLeft = icon('arrow-left', 14);
  svgUser      = icon('user', 16);
  svgUsers     = icon('users', 16);
  svgCpu       = icon('drama', 16);
  svgLock      = icon('lock', 14);
  svgEye       = icon('eye', 14);
  svgPencil    = icon('pencil', 14);
  renderNav();
  renderPage();
}

// ── Nav ───────────────────────────────────────────────────────────────

function renderNav() {
  const path = window.location.pathname;
  const adminSection = user.is_admin ? `
    <div class="snav-group-label">Admin</div>
    <a href="/settings/users" class="snav-item${path === '/settings/users' ? ' active' : ''}">
      ${svgUsers}
      Users
    </a>
  ` : '';

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
      ${svgCpu}
      Characters
    </a>
    ${adminSection}
  `;
}

// ── Account page ──────────────────────────────────────────────────────

function renderPage() {
  document.getElementById('smain').innerHTML = `
    <div class="smain-head">
      <h1>Account</h1>
      <p>Your profile and the keys to the kingdom.</p>
    </div>
    <div class="section">
      <h2>Profile</h2>
      <p class="lead">Your account on this server.</p>
      <div id="avatar-section"></div>
      <div class="field">
        <div class="meta">
          <span class="lbl">Username</span>
          <span class="desc">Contact an admin to change it.</span>
        </div>
        <div class="ctl">
          <span class="mono-tag">${escapeHtml(user.username)}</span>
        </div>
      </div>
      <div id="dn-section"></div>
    </div>
    <div class="section">
      <h2>Password</h2>
      <p class="lead">Protects your account when signing in.</p>
      <div id="pwd-section"></div>
    </div>
  `;
  const avatarUrl = `/api/users/${user.id}/avatar`;
  const displayUrl = user.has_avatar ? `${avatarUrl}?t=${Date.now()}` : avatarUrl;
  document.getElementById('avatar-section').innerHTML = renderAvatarSection(user.has_avatar, displayUrl);
  attachAvatarSection(document.getElementById('avatar-section'), {
    avatarUrl,
    onUpload: async (file) => {
      await auth.uploadAvatar(file);
      user = { ...user, has_avatar: true };
    },
    onDelete: async () => {
      await auth.deleteAvatar();
      user = { ...user, has_avatar: false };
    },
  });
  showDnDefault();
  showPwdDefault();
}

// ── Display name section ──────────────────────────────────────────────

function showDnDefault(successMsg) {
  const section = document.getElementById('dn-section');
  const current = user.display_name;
  section.innerHTML = `
    ${successMsg ? `<div class="field-msg field-msg--success">${escapeHtml(successMsg)}</div>` : ''}
    <div class="field">
      <div class="meta">
        <span class="lbl">Display name</span>
        <span class="desc">${current ? escapeHtml(current) : '<span style="color:var(--fg-faint)">Not set — username is shown instead.</span>'}</span>
      </div>
      <div class="ctl">
        <button class="btn btn-secondary btn-sm" id="open-dn-btn">
          ${svgPencil}
          Edit
        </button>
      </div>
    </div>
  `;
  document.getElementById('open-dn-btn').addEventListener('click', showDnForm);
}

function showDnForm() {
  const section = document.getElementById('dn-section');
  section.innerHTML = `
    <div class="pwd-form">
      <div class="pwd-row">
        <label for="dn-input">Display name</label>
        <input id="dn-input" type="text" class="input" placeholder="Your name" maxlength="100"
          value="${escapeHtml(user.display_name ?? '')}">
      </div>
      <div class="field-msg field-msg--error" id="dn-error" hidden></div>
      <div class="pwd-actions">
        <button class="btn btn-ghost btn-sm" id="cancel-dn-btn" type="button">Cancel</button>
        <button class="btn btn-primary btn-sm" id="save-dn-btn" type="button">Save</button>
      </div>
    </div>
  `;

  const input    = document.getElementById('dn-input');
  const errorEl  = document.getElementById('dn-error');
  const cancelBtn = document.getElementById('cancel-dn-btn');
  const saveBtn   = document.getElementById('save-dn-btn');

  input.focus();
  input.select();

  cancelBtn.addEventListener('click', () => showDnDefault());

  saveBtn.addEventListener('click', async () => {
    errorEl.hidden = true;
    saveBtn.disabled = true;
    saveBtn.textContent = 'Saving…';
    try {
      await auth.updateProfile(input.value.trim());
      user = { ...user, display_name: input.value.trim() || null };
      showDnDefault('Display name updated.');
    } catch (err) {
      saveBtn.disabled = false;
      saveBtn.textContent = 'Save';
      errorEl.hidden = false;
      errorEl.textContent = err.message;
    }
  });

  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') saveBtn.click();
    if (e.key === 'Escape') cancelBtn.click();
  });
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

  const newInput   = document.getElementById('pwd-new');
  const strengthEl = document.getElementById('pwd-strength');
  const fillEl     = document.getElementById('strength-fill');
  const labelEl    = document.getElementById('strength-label');
  const toggleBtn  = document.getElementById('pwd-toggle');
  const errorEl    = document.getElementById('pwd-error');
  const cancelBtn  = document.getElementById('cancel-pwd-btn');
  const saveBtn    = document.getElementById('save-pwd-btn');

  newInput.addEventListener('input', () => {
    if (!newInput.value) { strengthEl.hidden = true; return; }
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

    if (newPwd.length < 6) { showError('New password must be at least 6 characters.'); return; }
    if (newPwd !== confirmPwd) { showError('Passwords do not match.'); return; }

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

init();
