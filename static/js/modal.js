// Shared Escape handler — one listener for all modals, closes topmost open one.
const registry = [];

document.addEventListener('keydown', (e) => {
  if (e.key !== 'Escape') return;
  for (let i = registry.length - 1; i >= 0; i--) {
    if (registry[i].overlay.classList.contains('open')) {
      registry[i].close();
      break;
    }
  }
});

/**
 * Creates a modal scaffold: overlay → dialog, backdrop-click to close,
 * shared Escape handler, open/close methods.
 *
 * Callers set overlay.className and dialog.className, then populate dialog.
 */
export function createModal(ariaLabel) {
  const overlay = document.createElement('div');
  overlay.setAttribute('role', 'dialog');
  overlay.setAttribute('aria-modal', 'true');
  overlay.setAttribute('aria-label', ariaLabel);

  const dialog = document.createElement('div');
  overlay.appendChild(dialog);

  const modal = {
    overlay,
    dialog,
    open() { overlay.classList.add('open'); },
    close() { overlay.classList.remove('open'); },
  };

  overlay.addEventListener('mousedown', (e) => {
    if (e.target === overlay) modal.close();
  });

  registry.push(modal);
  document.body.appendChild(overlay);
  return modal;
}
