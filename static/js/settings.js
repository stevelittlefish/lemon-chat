import { auth, admin } from './api.js';

let user = null;
let currentPanel = 'account';

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
  const adminSection = user.is_admin ? `
    <div class="snav-group-label">Admin</div>
    <div class="snav-item ${currentPanel === 'users' ? 'active' : ''}" data-panel="users">
      ${svgUsers}
      Users
    </div>
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
    <div class="snav-item ${currentPanel === 'account' ? 'active' : ''}" data-panel="account">
      ${svgUser}
      Account
    </div>
    ${adminSection}
  `;

  document.querySelectorAll('.snav-item[data-panel]').forEach(el => {
    el.addEventListener('click', () => navigate(el.dataset.panel));
  });
}

function navigate(panel) {
  currentPanel = panel;
  renderNav();
  if (panel === 'account') showAccountPanel();
  else if (panel === 'users') showUsersPanel();
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

// ── Admin — Users panel ───────────────────────────────────────────────

let usersData = [];
let editingId = null;
let showingAddForm = false;

async function showUsersPanel() {
  editingId = null;
  showingAddForm = false;

  document.getElementById('smain').innerHTML = `
    <div class="smain-head">
      <h1>Users</h1>
      <p>Everyone who can sign in to this server.</p>
    </div>
    <div class="section">
      <div class="section-toolbar">
        <div>
          <h2>All users</h2>
          <p class="lead">Add or edit accounts. New users are created without a password.</p>
        </div>
        <button class="btn btn-secondary btn-sm" id="add-user-btn">
          ${svgPlus}
          Add user
        </button>
      </div>
      <div id="users-list"><div class="users-status">Loading…</div></div>
    </div>
  `;

  document.getElementById('add-user-btn').addEventListener('click', () => {
    showingAddForm = true;
    editingId = null;
    renderUsersList();
  });

  try {
    usersData = await admin.users.list();
  } catch {
    document.getElementById('users-list').innerHTML =
      `<div class="field-msg field-msg--error" style="padding:14px 0">Could not load users.</div>`;
    return;
  }
  renderUsersList();
}

function renderUsersList() {
  const list = document.getElementById('users-list');
  if (!list) return;

  let html = '';

  if (showingAddForm) {
    html += `
      <div class="user-row user-row--form">
        <input class="input user-name-input" id="add-username" placeholder="username" autocomplete="off" spellcheck="false">
        <label class="user-admin-check">
          <input type="checkbox" id="add-is-admin"> Admin
        </label>
        <div class="user-row-actions">
          <button class="btn btn-ghost btn-sm" id="cancel-add-btn">Cancel</button>
          <button class="btn btn-primary btn-sm" id="save-add-btn">Add user</button>
        </div>
      </div>
      <div class="user-form-err field-msg field-msg--error" id="add-err" hidden></div>`;
  }

  if (!usersData.length && !showingAddForm) {
    html += '<div class="users-status">No users yet.</div>';
  }

  for (const u of usersData) {
    if (editingId === u.id) {
      html += `
        <div class="user-row user-row--form" data-id="${u.id}">
          <input class="input user-name-input" id="edit-username-${u.id}" value="${escapeHtml(u.username)}" autocomplete="off" spellcheck="false">
          <label class="user-admin-check">
            <input type="checkbox" id="edit-is-admin-${u.id}" ${u.is_admin ? 'checked' : ''}> Admin
          </label>
          <div class="user-row-actions">
            <button class="btn btn-ghost btn-sm" data-action="cancel-edit">Cancel</button>
            <button class="btn btn-primary btn-sm" data-action="save-edit" data-id="${u.id}">Save</button>
          </div>
        </div>
        <div class="user-form-err field-msg field-msg--error" id="edit-err-${u.id}" hidden></div>`;
    } else {
      html += `
        <div class="user-row" data-id="${u.id}">
          <div class="user-name">${escapeHtml(u.username)}</div>
          ${u.is_admin ? '<span class="user-badge">admin</span>' : '<span></span>'}
          <div class="user-row-actions">
            <button class="btn btn-ghost btn-sm" data-action="edit" data-id="${u.id}">${svgPencil} Edit</button>
            <button class="btn btn-ghost btn-sm" data-action="delete" data-id="${u.id}">${svgTrash} Delete</button>
          </div>
        </div>`;
    }
  }

  list.innerHTML = html;

  if (showingAddForm) {
    const usernameInput = document.getElementById('add-username');
    document.getElementById('cancel-add-btn').addEventListener('click', () => {
      showingAddForm = false;
      renderUsersList();
    });
    document.getElementById('save-add-btn').addEventListener('click', createUser);
    usernameInput.addEventListener('keydown', e => {
      if (e.key === 'Enter') createUser();
      if (e.key === 'Escape') { showingAddForm = false; renderUsersList(); }
    });
    usernameInput.focus();
  }

  list.querySelectorAll('[data-action]').forEach(btn => {
    const action = btn.dataset.action;
    const id = Number(btn.dataset.id);
    switch (action) {
      case 'edit':
        btn.addEventListener('click', () => { editingId = id; showingAddForm = false; renderUsersList(); });
        break;
      case 'cancel-edit':
        btn.addEventListener('click', () => { editingId = null; renderUsersList(); });
        break;
      case 'save-edit':
        btn.addEventListener('click', () => saveEditUser(id));
        break;
      case 'delete':
        btn.addEventListener('click', () => deleteUser(id));
        break;
    }
  });

  if (editingId !== null) {
    const input = document.getElementById(`edit-username-${editingId}`);
    if (input) {
      input.focus();
      input.select();
      input.addEventListener('keydown', e => {
        if (e.key === 'Enter') saveEditUser(editingId);
        if (e.key === 'Escape') { editingId = null; renderUsersList(); }
      });
    }
  }
}

async function createUser() {
  const usernameInput = document.getElementById('add-username');
  const username = usernameInput.value.trim();
  const isAdmin = document.getElementById('add-is-admin').checked;
  const errorEl = document.getElementById('add-err');
  const saveBtn = document.getElementById('save-add-btn');

  errorEl.hidden = true;
  if (!username) {
    errorEl.hidden = false;
    errorEl.textContent = 'Username is required.';
    usernameInput.focus();
    return;
  }

  saveBtn.disabled = true;
  saveBtn.textContent = 'Adding…';

  try {
    const newUser = await admin.users.create({ username, is_admin: isAdmin });
    usersData.push(newUser);
    usersData.sort((a, b) => a.username.localeCompare(b.username));
    showingAddForm = false;
    renderUsersList();
  } catch (err) {
    saveBtn.disabled = false;
    saveBtn.textContent = 'Add user';
    errorEl.hidden = false;
    errorEl.textContent = err.message;
  }
}

async function saveEditUser(id) {
  const usernameInput = document.getElementById(`edit-username-${id}`);
  const username = usernameInput.value.trim();
  const isAdmin = document.getElementById(`edit-is-admin-${id}`).checked;
  const errorEl = document.getElementById(`edit-err-${id}`);
  const saveBtn = document.querySelector(`[data-action="save-edit"][data-id="${id}"]`);

  errorEl.hidden = true;
  if (!username) {
    errorEl.hidden = false;
    errorEl.textContent = 'Username is required.';
    usernameInput.focus();
    return;
  }

  saveBtn.disabled = true;
  saveBtn.textContent = 'Saving…';

  try {
    await admin.users.update(id, { username, is_admin: isAdmin });
    const u = usersData.find(u => u.id === id);
    if (u) { u.username = username; u.is_admin = isAdmin; }
    usersData.sort((a, b) => a.username.localeCompare(b.username));
    editingId = null;
    renderUsersList();
  } catch (err) {
    saveBtn.disabled = false;
    saveBtn.textContent = 'Save';
    errorEl.hidden = false;
    errorEl.textContent = err.message;
  }
}

async function deleteUser(id) {
  const u = usersData.find(u => u.id === id);
  if (!confirm(`Delete user "${u?.username}"? This cannot be undone.`)) return;

  try {
    await admin.users.delete(id);
    usersData = usersData.filter(u => u.id !== id);
    if (editingId === id) editingId = null;
    renderUsersList();
  } catch (err) {
    alert(`Could not delete user: ${err.message}`);
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

const svgUsers = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75"/></svg>`;

const svgLock = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V7a4 4 0 0 1 8 0v4"/></svg>`;

const svgEye = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>`;

const svgPlus = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 5v14M5 12h14"/></svg>`;

const svgPencil = `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>`;

const svgTrash = `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6M14 11v6"/><path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/></svg>`;

init();
