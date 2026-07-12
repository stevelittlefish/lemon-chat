const $ = (id) => document.getElementById(id);
const send = (message) => chrome.runtime.sendMessage(message);
const REDDIT_HOSTS = ['reddit.com', 'www.reddit.com', 'old.reddit.com', 'new.reddit.com', 'redd.it'];
let currentPageResult = null;

function isRedditURL(raw) {
  try { return REDDIT_HOSTS.includes(new URL(raw).hostname.toLowerCase()); }
  catch { return false; }
}

function loadMoreLimit() {
  const value = Number($('max-load-more').value);
  if (!Number.isInteger(value) || value < 0 || value > 500) throw new Error('Maximum load-more actions must be between 0 and 500');
  return value;
}

async function downloadResponse(response) {
  const url = URL.createObjectURL(new Blob([JSON.stringify(response, null, 2)], { type: 'application/json' }));
  await chrome.downloads.download({ url, filename: `reddit-import-${response.request_id}.json`, saveAs: true });
  setTimeout(() => URL.revokeObjectURL(url), 10000);
}

function validRequest(value) {
  if (value?.version !== 1 || !value.request_id || !Array.isArray(value.pages) || !value.pages.length) throw new Error('Expected a version 1 request bundle with request_id and pages');
  if (value.pages.length > 25) throw new Error('Request exceeds the 25-page limit');
  for (const page of value.pages) {
    const url = new URL(page.url);
    if (!REDDIT_HOSTS.includes(url.hostname.toLowerCase())) throw new Error(`Unsupported Reddit host: ${url.hostname}`);
  }
  return value;
}

async function refresh() {
  const current = await send({ type: 'getState' });
  const total = current.request?.pages?.length || 1;
  $('progress').max = total;
  $('progress').value = current.index || 0;
  $('status').textContent = `${current.status || 'idle'} — ${current.index || 0} of ${current.request?.pages?.length || 0}${current.currentURL ? `\n${current.currentURL}` : ''}`;
  if (currentPageResult && !current.running) $('status').textContent = `${currentPageResult.comments} comments captured`;
  const complete = current.status === 'complete' || (currentPageResult && !current.running);
  $('capture-status').classList.toggle('complete', complete);
  $('status-heading').textContent = currentPageResult && !current.running ? 'Current page exported' : complete ? 'Capture complete' : current.running ? 'Capture in progress' : current.status === 'stopped' ? 'Capture stopped' : 'Ready to capture';
  $('export').disabled = !current.results?.length;
  $('stop').disabled = !current.running;
  $('skip').disabled = !current.running;
  $('retry').disabled = current.running || !current.results?.length;
}

$('start').addEventListener('click', async () => {
  $('error').textContent = '';
  try {
    currentPageResult = null;
    const request = validRequest(JSON.parse($('request').value));
    const maxLoadMore = loadMoreLimit();
    await chrome.storage.local.set({ lastRequest: $('request').value, maxLoadMore });
    await send({ type: 'start', request, delayMs: Number($('delay').value) * 1000, maxLoadMore });
    refresh();
  } catch (error) { $('error').textContent = error.message; }
});

$('stop').addEventListener('click', () => send({ type: 'stop' }).then(refresh));
$('skip').addEventListener('click', () => send({ type: 'skip' }).then(refresh));
$('retry').addEventListener('click', () => send({ type: 'retry' }).then(refresh));
$('export').addEventListener('click', async () => {
  const current = await send({ type: 'getState' });
  const response = {
    version: current.request.version,
    request_id: current.request.request_id,
    captured_at: new Date().toISOString(),
    pages: current.results,
  };
  await downloadResponse(response);
});

$('export-current').addEventListener('click', async () => {
  $('error').textContent = '';
  try {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    if (!tab?.id || !isRedditURL(tab.url)) throw new Error('The active tab is not a supported Reddit page');
    const maxLoadMore = loadMoreLimit();
    await chrome.storage.local.set({ maxLoadMore });
    $('export-current').disabled = true;
    $('export-current').textContent = 'Capturing…';
    const page = { url: tab.url };
    const result = await chrome.tabs.sendMessage(tab.id, { type: 'capture', page, limits: { max_expand_actions: maxLoadMore } });
    const requestID = `current-${Date.now()}`;
    await downloadResponse({ version: 1, request_id: requestID, captured_at: new Date().toISOString(), pages: [result] });
    currentPageResult = { comments: result.comments?.length || 0 };
    await refresh();
  } catch (error) {
    $('error').textContent = error.message;
  } finally {
    $('export-current').disabled = false;
    $('export-current').textContent = 'Export current page';
  }
});

const { lastRequest = '', maxLoadMore = 5 } = await chrome.storage.local.get(['lastRequest', 'maxLoadMore']);
$('request').value = lastRequest;
$('max-load-more').value = maxLoadMore;
const [activeTab] = await chrome.tabs.query({ active: true, currentWindow: true });
$('current-page').hidden = !isRedditURL(activeTab?.url);
await refresh();
setInterval(refresh, 750);
