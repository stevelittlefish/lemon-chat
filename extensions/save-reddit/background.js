const STATE_KEY = 'captureState';
const CLEAR_DONE_BADGE_ALARM = 'clearDoneBadge';

async function state() {
  return (await chrome.storage.local.get(STATE_KEY))[STATE_KEY] || { status: 'idle' };
}

async function save(next) {
  await chrome.storage.local.set({ [STATE_KEY]: next });
  return next;
}

const delay = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

async function capturePage(tabId, page, limits) {
  for (let attempt = 0; attempt < 20; attempt += 1) {
    try {
      return await chrome.tabs.sendMessage(tabId, { type: 'capture', page, limits });
    } catch {
      await delay(500);
    }
  }
  throw new Error('The Reddit page did not become ready for capture');
}

async function run() {
  let current = await state();
  if (current.running) return;
  current.running = true;
  await save(current);

  try {
    while (current.index < current.request.pages.length) {
      current = await state();
      if (current.stopRequested) {
        current.status = 'stopped';
        break;
      }
      if (current.skipRequested) {
        current.results.push(failedPage(current.request.pages[current.index], 'Skipped by user'));
        current.index += 1;
        current.skipRequested = false;
        await save(current);
        continue;
      }

      const page = current.request.pages[current.index];
      current.status = 'capturing';
      current.currentURL = page.url;
      await save(current);
      let tab;
      try {
        tab = await chrome.tabs.create({ url: page.url, active: false });
        current.currentTabId = tab.id;
        await save(current);
        const requestLimit = current.request.limits?.max_expand_actions ?? 500;
        const limits = { ...(current.request.limits || {}), max_expand_actions: Math.min(requestLimit, current.maxLoadMore ?? 5) };
        const result = await capturePage(tab.id, page, limits);
        current = await state();
        if (current.stopRequested) current.results.push(failedPage(page, 'Capture stopped by user'));
        else if (current.skipRequested) current.results.push(failedPage(page, 'Skipped by user'));
        else current.results.push(result);
      } catch (error) {
        current = await state();
        if (current.stopRequested) current.results.push(failedPage(page, 'Capture stopped by user'));
        else if (current.skipRequested) current.results.push(failedPage(page, 'Skipped by user'));
        else current.results.push(failedPage(page, error.message));
      } finally {
        if (tab?.id) await chrome.tabs.remove(tab.id).catch(() => {});
      }
      current.index += 1;
      current.currentURL = '';
      current.currentTabId = null;
      current.skipRequested = false;
      await save(current);
      if (current.index < current.request.pages.length) await delay(current.delayMs);
    }

    current = await state();
    if (current.stopRequested) {
      while (current.index < current.request.pages.length) {
        current.results.push(failedPage(current.request.pages[current.index], 'Capture stopped by user'));
        current.index += 1;
      }
    }
    if (!current.stopRequested && current.index >= current.request.pages.length) {
      current.status = 'complete';
      await chrome.action.setBadgeBackgroundColor({ color: '#5c7a3e' });
      await chrome.action.setBadgeText({ text: 'Done' });
      await chrome.alarms.create(CLEAR_DONE_BADGE_ALARM, { delayInMinutes: 1 });
    }
  } finally {
    current.running = false;
    await save(current);
  }
}

function failedPage(page, failure) {
  return {
    requested_url: page.url,
    canonical_url: page.url,
    post: {}, comments: [], complete: false,
    warnings: [], failure,
  };
}

chrome.runtime.onMessage.addListener((message, _sender, respond) => {
  (async () => {
    if (message.type === 'start') {
      await chrome.alarms.clear(CLEAR_DONE_BADGE_ALARM);
      await chrome.action.setBadgeText({ text: '' });
      await save({
        status: 'queued', running: false, stopRequested: false, skipRequested: false,
        request: message.request, results: [], index: 0,
        delayMs: Math.max(500, Math.min(30000, message.delayMs || 1500)),
        maxLoadMore: Math.max(0, Math.min(500, message.maxLoadMore ?? 5)),
      });
      run();
    } else if (message.type === 'stop') {
      const current = await state(); current.stopRequested = true; await save(current);
      if (current.currentTabId) await chrome.tabs.remove(current.currentTabId).catch(() => {});
    } else if (message.type === 'skip') {
      const current = await state(); current.skipRequested = true; await save(current);
      if (current.currentTabId) await chrome.tabs.remove(current.currentTabId).catch(() => {});
    } else if (message.type === 'retry') {
      const current = await state();
      if (!current.running && current.results.length) {
        current.index = Math.max(0, current.index - 1);
        current.results.pop();
        current.status = 'queued'; current.stopRequested = false;
        await save(current); run();
      }
    } else if (message.type === 'getState') {
      respond(await state()); return;
    }
    respond({ ok: true });
  })();
  return true;
});

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === CLEAR_DONE_BADGE_ALARM) chrome.action.setBadgeText({ text: '' });
});
