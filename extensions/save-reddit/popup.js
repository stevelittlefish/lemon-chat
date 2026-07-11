const $ = (id) => document.getElementById(id);
const send = (message) => chrome.runtime.sendMessage(message);

function validRequest(value) {
  if (value?.version !== 1 || !value.request_id || !Array.isArray(value.pages) || !value.pages.length) throw new Error('Expected a version 1 request bundle with request_id and pages');
  if (value.pages.length > 25) throw new Error('Request exceeds the 25-page limit');
  for (const page of value.pages) {
    const url = new URL(page.url);
    if (!['reddit.com', 'www.reddit.com', 'old.reddit.com', 'new.reddit.com', 'redd.it'].includes(url.hostname.toLowerCase())) throw new Error(`Unsupported Reddit host: ${url.hostname}`);
  }
  return value;
}

async function refresh() {
  const current = await send({ type: 'getState' });
  const total = current.request?.pages?.length || 1;
  $('progress').max = total;
  $('progress').value = current.index || 0;
  $('status').textContent = `${current.status || 'idle'} — ${current.index || 0} of ${current.request?.pages?.length || 0}${current.currentURL ? `\n${current.currentURL}` : ''}`;
  $('export').disabled = !current.results?.length;
  $('stop').disabled = !current.running;
  $('skip').disabled = !current.running;
  $('retry').disabled = current.running || !current.results?.length;
}

$('start').addEventListener('click', async () => {
  $('error').textContent = '';
  try {
    const request = validRequest(JSON.parse($('request').value));
    await chrome.storage.local.set({ lastRequest: $('request').value });
    await send({ type: 'start', request, delayMs: Number($('delay').value) * 1000 });
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
  const url = URL.createObjectURL(new Blob([JSON.stringify(response, null, 2)], { type: 'application/json' }));
  await chrome.downloads.download({ url, filename: `reddit-import-${response.request_id}.json`, saveAs: true });
  setTimeout(() => URL.revokeObjectURL(url), 10000);
});

const { lastRequest = '' } = await chrome.storage.local.get('lastRequest');
$('request').value = lastRequest;
await refresh();
setInterval(refresh, 750);
