export function escapeHtml(str) {
  return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

// Formats a model's per-token prices as a compact "in / out per 1M" rate, e.g.
// "$3.00 / $15.00 per 1M". Returns '' when the model has no known price.
export function formatModelRate(model) {
  if (model?.prompt_usd == null || model?.completion_usd == null) return '';
  const per1M = (v) => {
    const n = v * 1e6;
    return n >= 1 ? `$${n.toFixed(2)}` : `$${n.toFixed(3)}`;
  };
  return `${per1M(model.prompt_usd)} / ${per1M(model.completion_usd)} per 1M`;
}

export async function copyToClipboard(text) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch {
      // Fall through for denied permissions and browsers with partial support.
    }
  }

  const textarea = document.createElement('textarea');
  textarea.value = text;
  textarea.setAttribute('readonly', '');
  textarea.style.cssText = 'position:fixed;opacity:0;top:0;left:0';
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  const copied = document.execCommand('copy');
  textarea.remove();
  if (!copied) throw new Error('The browser could not copy to the clipboard.');
}
