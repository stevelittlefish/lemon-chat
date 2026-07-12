const text = (element) => element?.innerText?.trim() || '';
const attr = (element, name) => element?.getAttribute(name) || '';
const first = (...selectors) => selectors.map((selector) => document.querySelector(selector)).find(Boolean);
const all = (...selectors) => [...new Set(selectors.flatMap((selector) => [...document.querySelectorAll(selector)]))];

function scoreValue(raw) {
  const match = String(raw || '').replace(/,/g, '').match(/-?\d+(?:\.\d+)?/);
  if (!match) return undefined;
  let value = Number(match[0]);
  if (/k/i.test(raw)) value *= 1000;
  return Math.round(value);
}

function canonicalURL() {
  const canonical = document.querySelector('link[rel="canonical"]')?.href || location.href;
  const url = new URL(canonical);
  url.search = ''; url.hash = '';
  return url.toString();
}

function absoluteURL(raw) {
  if (!raw) return '';
  try { return new URL(raw, location.origin).toString(); }
  catch { return ''; }
}

async function expandAndScroll(limits, warnings) {
  const deadline = Date.now() + Math.min(300, limits.max_seconds_per_page || 300) * 1000;
  const maxActions = Math.max(0, Math.min(500, limits.max_expand_actions ?? 500));
  let actions = 0;
  while (actions < maxActions && Date.now() < deadline) {
    const buttons = all('button', 'a').filter((el) => {
      const label = `${text(el)} ${attr(el, 'aria-label')}`.toLowerCase();
      const href = attr(el, 'href');
      const navigates = el.tagName === 'A' && href && !href.startsWith('#') && !href.toLowerCase().startsWith('javascript:');
      return /more replies|continue this thread|view more comments|load more comments/.test(label) && !navigates && el.offsetParent !== null;
    });
    if (!buttons.length) break;
    for (const button of buttons.slice(0, maxActions - actions)) {
      button.click(); actions += 1; await new Promise((resolve) => setTimeout(resolve, 250));
    }
    window.scrollTo(0, document.body.scrollHeight);
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  if (actions >= maxActions) warnings.push('Capture stopped at the configured expansion limit');
  if (Date.now() >= deadline) warnings.push('Capture stopped at the configured time limit');
}

function capturePost() {
  const post = first('shreddit-post', '[data-test-id="post-content"]', '.thing.link');
  const body = first('shreddit-post [slot="text-body"]', '[data-test-id="post-content"] [data-click-id="text"]', '.thing.link .usertext-body');
  const author = attr(post, 'author') || text(first('shreddit-post [slot="authorName"]', '[data-test-id="post-content"] a[href*="/user/"]', '.thing.link .author'));
  const score = scoreValue(attr(post, 'score') || text(first('shreddit-post [slot="vote-button"]', '[data-test-id="post-content"] [id*="vote-arrows"]', '.thing.link .score')));
  return { author, body: text(body), permalink: canonicalURL(), ...(score === undefined ? {} : { score }) };
}

function captureComments(limit) {
  const nodes = all('shreddit-comment', '[data-testid="comment"]', '.thing.comment');
  return nodes.slice(0, limit).map((node) => {
    const body = text(firstWithin(node, '[slot="comment"]', '[data-testid="comment"] p', '.md'));
    const author = attr(node, 'author') || text(firstWithin(node, '[slot="authorName"]', 'a[href*="/user/"]', '.author'));
    const permalink = absoluteURL(attr(node, 'permalink') || firstWithin(node, 'a[data-testid="comment_timestamp"]', 'a.bylink')?.href || '');
    const depthRaw = attr(node, 'depth') || node.dataset?.depth || '0';
    const score = scoreValue(attr(node, 'score') || text(firstWithin(node, '[slot="vote-button"]', '.score')));
    return { author, body, permalink, depth: Math.max(0, Number.parseInt(depthRaw, 10) || 0), ...(score === undefined ? {} : { score }) };
  }).filter((comment) => comment.body);
}

function firstWithin(root, ...selectors) {
  return selectors.map((selector) => root?.querySelector(selector)).find(Boolean);
}

chrome.runtime.onMessage.addListener((message, _sender, respond) => {
  if (message.type !== 'capture') return;
  (async () => {
    const warnings = [];
    await expandAndScroll(message.limits || {}, warnings);
    const post = capturePost();
    const maxComments = Math.min(2000, message.limits?.max_comments || 2000);
    const comments = captureComments(maxComments);
    if (comments.length >= maxComments) warnings.push('Capture reached the configured comment limit');
    if (!post.body && !comments.length) throw new Error('No rendered post or comments were recognized; the page may be private, gated, deleted, rate-limited, or Reddit may have changed its markup');
    if (/quarantined|over 18|mature content|private community/i.test(document.body.innerText)) warnings.push('The page displayed an access or content gate');
    respond({
      requested_url: message.page.url,
      canonical_url: canonicalURL(),
      title: text(first('h1', 'shreddit-title')) || document.title.replace(/\s*: r\/.*$/i, ''),
      subreddit: attr(first('shreddit-post'), 'subreddit-prefixed-name').replace(/^r\//, '') || location.pathname.split('/')[2] || '',
      post, comments, complete: warnings.length === 0,
      warnings,
    });
  })().catch((error) => respond({
    requested_url: message.page.url, canonical_url: canonicalURL(), post: {}, comments: [],
    complete: false, warnings: [], failure: error.message,
  }));
  return true;
});
