const _cache = new Map();

const ICONS = [
  'plus', 'trash', 'settings', 'log-out', 'send',
  'cpu', 'drama', 'chevron-down', 'arrow-left', 'user', 'users',
  'lock', 'eye', 'pencil', 'check', 'info', 'x', 'code',
  'ellipsis-vertical', 'refresh-cw', 'copy', 'fork', 'list',
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
