// Lightweight markdown renderer for chat messages.
// Handles: paragraphs, **bold**, `inline code`, and ```fenced code blocks```.

export function render(text) {
  const html = [];
  const lines = text.split('\n');
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];

    // Fenced code block
    if (line.startsWith('```')) {
      const lang = line.slice(3).trim();
      const code = [];
      i++;
      while (i < lines.length && !lines[i].startsWith('```')) {
        code.push(lines[i]);
        i++;
      }
      html.push(`<pre><code${lang ? ` class="language-${escape(lang)}"` : ''}>${escape(code.join('\n'))}</code></pre>`);
      i++;
      continue;
    }

    // Blank line — paragraph break (skip)
    if (line.trim() === '') {
      i++;
      continue;
    }

    // Collect non-blank lines into a paragraph
    const para = [];
    while (i < lines.length && lines[i].trim() !== '' && !lines[i].startsWith('```')) {
      para.push(lines[i]);
      i++;
    }
    html.push(`<p>${inlineMarkdown(para.join('\n'))}</p>`);
  }

  return html.join('');
}

function inlineMarkdown(text) {
  return escape(text)
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/`([^`]+)`/g, '<code>$1</code>');
}

function escape(text) {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}
