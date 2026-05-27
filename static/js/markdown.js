import { marked } from './vendor/marked.esm.js';

marked.setOptions({ breaks: true });

export function render(text) {
  return marked.parse(text);
}
