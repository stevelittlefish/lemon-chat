// Shared avatar upload UI for the settings pages (account + character edit).

const DEFAULT_PLACEHOLDER = `<svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="8" r="4"/><path d="M4 20c0-4 3.6-7 8-7s8 3 8 7"/></svg>`;

export function renderAvatarSection(hasAvatar, avatarUrl, placeholderIcon = DEFAULT_PLACEHOLDER) {
  const preview = hasAvatar
    ? `<img class="avatar-img" src="${escapeAttr(avatarUrl)}" alt="">`
    : `<div class="avatar-placeholder">${placeholderIcon}</div>`;

  return `
    <div class="avatar-section">
      ${preview}
      <div class="avatar-edit">
        <label class="btn btn-secondary btn-sm avatar-upload-label">
          ${hasAvatar ? 'Change photo' : 'Upload photo'}
          <input type="file" accept="image/*" class="avatar-file-input" style="display:none">
        </label>
        ${hasAvatar ? `<button type="button" class="avatar-remove-btn">Remove photo</button>` : ''}
        <p class="avatar-hint">JPG or PNG, up to 5 MB.</p>
      </div>
      <div class="field-msg field-msg--error avatar-err" hidden></div>
    </div>
  `;
}

// opts: { onUpload(file): Promise, onDelete(): Promise, avatarUrl: string, placeholderIcon?: string }
export function attachAvatarSection(container, opts) {
  const { onUpload, onDelete, avatarUrl, placeholderIcon } = opts;
  const fileInput = container.querySelector('.avatar-file-input');
  const removeBtn = container.querySelector('.avatar-remove-btn');
  const errEl     = container.querySelector('.avatar-err');

  if (fileInput) {
    fileInput.addEventListener('change', async () => {
      const file = fileInput.files[0];
      if (!file) return;
      if (errEl) errEl.hidden = true;
      try {
        await onUpload(file);
        const freshUrl = avatarUrl + '?t=' + Date.now();
        container.innerHTML = renderAvatarSection(true, freshUrl, placeholderIcon);
        attachAvatarSection(container, opts);
      } catch (err) {
        if (errEl) { errEl.hidden = false; errEl.textContent = err.message; }
      }
    });
  }

  if (removeBtn) {
    removeBtn.addEventListener('click', async () => {
      if (errEl) errEl.hidden = true;
      try {
        await onDelete();
        container.innerHTML = renderAvatarSection(false, '', placeholderIcon);
        attachAvatarSection(container, opts);
      } catch (err) {
        if (errEl) { errEl.hidden = false; errEl.textContent = err.message; }
      }
    });
  }
}

function escapeAttr(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;');
}
