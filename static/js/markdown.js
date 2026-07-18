import { marked } from './vendor/marked.esm.js';
import katex from './vendor/katex.esm.js';
import DOMPurify from './vendor/dompurify.esm.js';

marked.setOptions({ breaks: true });

marked.use({
  extensions: [
    {
      name: 'blockMath',
      level: 'block',
      start(src) { return src.indexOf('$$'); },
      tokenizer(src) {
        const match = src.match(/^\$\$([\s\S]+?)\$\$/);
        if (match) return { type: 'blockMath', raw: match[0], text: match[1].trim() };
      },
      renderer(token) {
        return katex.renderToString(token.text, { displayMode: true, throwOnError: false });
      },
    },
    {
      name: 'inlineMath',
      level: 'inline',
      start(src) { return src.indexOf('$'); },
      tokenizer(src) {
        const match = src.match(/^\$([^\$\n]+?)\$/);
        if (match) return { type: 'inlineMath', raw: match[0], text: match[1].trim() };
      },
      renderer(token) {
        return katex.renderToString(token.text, { displayMode: false, throwOnError: false });
      },
    },
  ],
});

export function render(text) {
  const div = document.createElement('div');
  div.innerHTML = DOMPurify.sanitize(marked.parse(text), {
    FORBID_TAGS: ['style', 'marquee', 'blink', 'frame', 'frameset', 'iframe', 'object', 'embed', 'applet', 'link', 'meta'],
  });
  // All links in rendered content are external — open them in a new tab.
  div.querySelectorAll('a').forEach((a) => {
    a.setAttribute('target', '_blank');
    a.setAttribute('rel', 'noopener');
    if (/^S\d+$/.test(a.textContent.trim())) {
      a.classList.add('source-citation');
    }
  });
  return div.innerHTML;
}
