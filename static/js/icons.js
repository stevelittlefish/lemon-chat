const _cache = new Map();

const ICONS = [
  'plus', 'trash', 'settings', 'log-out', 'send', 'square',
  'cpu', 'drama', 'chevron-down', 'chevron-left', 'chevron-right', 'arrow-left', 'user', 'users',
  'lock', 'lock-open', 'eye', 'pencil', 'check', 'info', 'x', 'code',
  'ellipsis-vertical', 'refresh-cw', 'copy', 'fork', 'list',
  'layout-grid',
  'play', 'rotate-ccw', 'rotate-cw', 'sliders', 'save',
  'globe',
  'file-text', 'file-code', 'file', 'download', 'image',
  'star',
];

export async function preload() {
  await Promise.all(ICONS.map(async name => {
    const r = await fetch(`/assets/icons/${name}.svg`);
    _cache.set(name, (await r.text()).trim());
  }));
}

export function icon(name, size) {
  let svg = _cache.get(name) ?? '';
  if (size) svg = svg.replace('<svg ', `<svg width="${size}" height="${size}" `);
  return svg;
}
