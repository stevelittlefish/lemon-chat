// Shared avatar upload UI for the settings pages (account + character edit).

export function renderAvatarSection(hasAvatar, avatarUrl) {
  return `
    <div class="avatar-section">
      ${hasAvatar ? `<div class="avatar-lg"><img src="${escapeAttr(avatarUrl)}" alt=""></div>` : ''}
      <div class="avatar-actions">
        <label class="btn btn-secondary btn-sm avatar-upload-label">
          ${hasAvatar ? 'Change photo' : 'Upload photo'}
          <input type="file" accept="image/*" class="avatar-file-input" style="display:none">
        </label>
        ${hasAvatar ? `<button type="button" class="avatar-remove-btn">Remove</button>` : ''}
      </div>
      <div class="field-msg field-msg--error avatar-err" hidden></div>
    </div>
  `;
}

// Attaches upload/delete event handlers to a rendered avatar section.
// opts: { onUpload(file): Promise, onDelete(): Promise, avatarUrl: string }
export function attachAvatarSection(container, opts) {
  const { onUpload, onDelete, avatarUrl } = opts;
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
        // Cache-bust by appending a timestamp to force the browser to re-fetch.
        const freshUrl = avatarUrl + '?t=' + Date.now();
        container.innerHTML = renderAvatarSection(true, freshUrl);
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
        container.innerHTML = renderAvatarSection(false, '');
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
