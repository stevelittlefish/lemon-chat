// Research page: start research jobs, watch live progress, read reports.
import { requireAuth } from './settings-auth.js';
import { research, models } from './api.js';
import { render } from './markdown.js';
import { copyToClipboard, escapeHtml } from './utils.js';

const main = document.getElementById('research-main');

let modelList = [];
let formDefaults = { effort: 3, max_time_minutes: 10 };
let abortEvents = null; // aborts the active SSE subscription

// Effort levels — value matches the 1–5 scale the backend expects.
const EFFORT_LEVELS = [
  [1, 'half arsed — one round'],
  [2, 'low — two rounds'],
  [3, 'default'],
  [4, 'extra — one bonus round'],
  [5, 'too much — two bonus rounds'],
];

const EFFORT_NAMES = { 1: 'half arsed', 2: 'low', 3: 'default', 4: 'extra', 5: 'too much' };

function stopEvents() {
  if (abortEvents) { abortEvents(); abortEvents = null; }
}

function setBackBtn(toList) {
  const btn = document.getElementById('back-to-menu');
  if (!btn) return;
  if (toList) {
    btn.href = '#';
    btn.onclick = (e) => { e.preventDefault(); location.hash = ''; };
    document.getElementById('back-label').textContent = 'research';
  } else {
    btn.href = '/menu';
    btn.onclick = null;
    document.getElementById('back-label').textContent = 'menu';
  }
}

function formatDuration(ms) {
  const s = Math.round(ms / 1000);
  if (s < 60) return `${s}s`;
  return `${Math.floor(s / 60)}m ${s % 60}s`;
}

function formatDate(iso) {
  const d = new Date(iso);
  return d.toLocaleDateString(undefined, { day: 'numeric', month: 'short' }) + ' ' +
    d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
}

function statusBadge(status) {
  switch (status) {
    case 'running':
    case 'pending': return '<span class="badge live"><span class="dot"></span>running</span>';
    case 'done': return '<span class="badge accent"><span class="dot"></span>done</span>';
    case 'error': return '<span class="badge warn"><span class="dot"></span>error</span>';
    case 'cancelled': return '<span class="badge"><span class="dot"></span>cancelled</span>';
    case 'awaiting_reddit': return '<span class="badge warn"><span class="dot"></span>awaiting Reddit</span>';
    default: return `<span class="badge">${escapeHtml(status)}</span>`;
  }
}

// ── List view ───────────────────────────────────────────────

async function showList() {
  stopEvents();
  setBackBtn(false);
  const jobs = await research.list();

  const options = ['<option value="">default model</option>']
    .concat(modelList.map((m) =>
      `<option value="${escapeHtml(m.name)}">${escapeHtml(m.display_name || m.name)}</option>`))
    .join('');

  const effortOptions = EFFORT_LEVELS.map(([value, label]) =>
    `<option value="${value}"${value === formDefaults.effort ? ' selected' : ''}>${label}</option>`)
    .join('');

  const items = jobs.map((j) => `
    <div class="card card-interactive research-item" data-id="${j.id}">
      <span class="research-item-query">${escapeHtml(j.title || j.query)}</span>
      <span class="research-item-details">
        <span class="research-item-model">${escapeHtml(j.model)}</span>
        <span class="research-item-meta">${formatDate(j.created_at)}</span>
        ${statusBadge(j.status)}
      </span>
    </div>`).join('');

  main.innerHTML = `
    <div class="card research-form">
      <div class="research-form-heading">
        <h2 class="research-form-title">New Research</h2>
        <a class="research-help-link" href="/research/help" target="_blank" rel="noopener noreferrer">help</a>
      </div>
      <input id="research-title" class="input" type="text" placeholder="title (optional)">
      <textarea id="research-query" class="input textarea" rows="3"
        placeholder="research question or prompt (required if no title)"></textarea>
      <div class="research-form-row">
        <select id="research-model" class="input">${options}</select>
        <label class="research-field">mode
          <select id="research-mode" class="input">
            <option value="research" selected>research</option>
            <option value="brainstorm">brainstorm</option>
          </select>
        </label>
        <label class="research-field research-check research-force-search" id="research-force-search-field" hidden>
          <input type="checkbox" id="research-force-search"> always search the web
        </label>
        <label class="research-field research-check">
          <input type="checkbox" id="research-deep-report"> in-depth report
        </label>
        <label class="research-field">effort
          <select id="research-effort" class="input">${effortOptions}</select>
        </label>
        <label class="research-field">time limit
          <input id="research-time" class="input research-time-input" type="number" min="1" max="240"
            value="${formDefaults.max_time_minutes}"> min
        </label>
        <button id="research-start" class="btn btn-primary">start research</button>
      </div>
      <div class="research-form-options">
        <label class="research-field research-check">
          <input type="checkbox" id="research-pause-reddit"> pause to import Reddit results
        </label>
      </div>
    </div>
    <p class="research-section-label">past research</p>
    <div class="research-list">
      ${items || '<p class="research-empty">no research yet — ask a question above to get started</p>'}
    </div>`;

  // Relabel the prompt and action when switching to brainstorm mode, which is
  // about inventing/designing rather than searching the web.
  const modeSel = document.getElementById('research-mode');
  modeSel.addEventListener('change', () => {
    const brainstorm = modeSel.value === 'brainstorm';
    document.getElementById('research-query').placeholder = brainstorm
      ? 'what to brainstorm or design (required if no title)'
      : 'research question or prompt (required if no title)';
    document.getElementById('research-start').textContent = brainstorm ? 'start brainstorm' : 'start research';
    // The force-search toggle only changes brainstorm behaviour — research mode
    // always searches — so it is only shown for brainstorm.
    document.getElementById('research-force-search-field').hidden = !brainstorm;
  });

  document.getElementById('research-start').addEventListener('click', startResearch);
  document.getElementById('research-query').addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) startResearch();
  });
  document.getElementById('research-title').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') document.getElementById('research-query').focus();
  });
  main.querySelectorAll('.research-item').forEach((el) => {
    el.addEventListener('click', () => { location.hash = el.dataset.id; });
  });
}

async function startResearch() {
  const title = document.getElementById('research-title').value.trim();
  const query = document.getElementById('research-query').value.trim();
  if (!title && !query) return;
  const model = document.getElementById('research-model').value;
  const mode = document.getElementById('research-mode').value;
  const forceSearch = mode === 'brainstorm' && document.getElementById('research-force-search').checked;
  const deepReport = document.getElementById('research-deep-report').checked;
  const pauseRedditImport = document.getElementById('research-pause-reddit').checked;
  const effort = parseInt(document.getElementById('research-effort').value, 10) || formDefaults.effort;
  const maxTimeMinutes = parseInt(document.getElementById('research-time').value, 10) || formDefaults.max_time_minutes;
  const btn = document.getElementById('research-start');
  btn.disabled = true;
  try {
    const job = await research.start(title, query, model, mode, forceSearch, deepReport, pauseRedditImport, effort, maxTimeMinutes);
    location.hash = job.id;
  } catch (err) {
    btn.disabled = false;
    alert(err.message);
  }
}

// ── Downloads ───────────────────────────────────────────────

function slugify(text) {
  return text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 60) || 'research';
}

function downloadFile(filename, mime, content) {
  const url = URL.createObjectURL(new Blob([content], { type: mime }));
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

// Wraps the rendered report in a self-contained HTML document styled to
// match the design system (paper background, warm ink, serif headings).
function htmlDocument(title, bodyHtml) {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>${escapeHtml(title)}</title>
<style>
  body { background: #FBF7EC; color: #1B1814; font-family: 'Source Sans 3', ui-sans-serif, system-ui, sans-serif;
         font-size: 16px; line-height: 1.6; max-width: 760px; margin: 0 auto; padding: 48px 24px 96px; }
  h1, h2, h3 { font-family: 'Source Serif 4', ui-serif, Georgia, serif; line-height: 1.25; }
  h2 { border-bottom: 1px solid #E7DFC9; padding-bottom: 8px; margin-top: 40px; }
  a { color: #3565A8; }
  code, pre { font-family: 'JetBrains Mono', ui-monospace, monospace; font-size: 0.92em; }
  code:not(pre code) { background: #F6EFDD; border: 1px solid #E7DFC9; border-radius: 2px; padding: 1px 6px; }
  pre { background: #F6EFDD; border: 1px solid #E7DFC9; border-radius: 6px; padding: 12px 16px; overflow-x: auto; }
  table { border-collapse: collapse; margin: 16px 0; font-size: 14px; }
  th, td { border: 1px solid #E7DFC9; padding: 6px 12px; text-align: left; }
  th { background: #F6EFDD; }
  blockquote { border-left: 2px solid #D6CDB7; margin-left: 0; padding-left: 16px; color: #6B6357; }
  details { border: 1px solid #E7DFC9; border-radius: 6px; padding: 8px 16px; margin: 16px 0; }
  details summary { cursor: pointer; }
  hr { border: none; border-top: 1px solid #E7DFC9; }
</style>
</head>
<body>
<article>
${bodyHtml}
</article>
</body>
</html>
`;
}

// ── Detail view ─────────────────────────────────────────────

function progressLine(ev) {
  const host = (url) => { try { return new URL(url).hostname; } catch { return url; } };
  switch (ev.phase) {
    case 'planning': return ev.message || 'planning research approach';
    case 'searching': return `round ${ev.round} — searching: ${(ev.queries || []).join(' · ')}`;
    case 'reading': return `round ${ev.round} — reading ${host(ev.url)}`;
    case 'analyzing': {
      const n = ev.total_findings || 0;
      return `round ${ev.round} — synthesizing (${n} finding${n === 1 ? '' : 's'})`;
    }
    case 'deciding': return `round ${ev.round} — stop check: ${ev.message}`;
    case 'writing': return ev.message || 'writing final report';
    case 'note': return ev.message;
    case 'warning': return ev.message;
    default: return ev.message || ev.phase;
  }
}

// Live preview of text being generated during synthesis / report writing.
// Created on the first streaming event, updated in place, removed when the
// generation phase ends.
function updateStream(ev) {
  let box = document.getElementById('research-stream');
  if (!box) {
    const log = document.getElementById('research-progress');
    if (!log) return;
    box = document.createElement('div');
    box.id = 'research-stream';
    box.className = 'research-stream';
    box.innerHTML = `
      <div class="research-stream-label"></div>
      <div class="research-stream-text"><span class="research-stream-tail"></span><span class="research-stream-caret"></span></div>`;
    log.after(box);
  }
  const verb = ev.phase === 'writing' ? 'writing report' : 'synthesizing';
  box.querySelector('.research-stream-label').textContent =
    `${verb} · ${ev.generated.toLocaleString()} chars`;
  box.querySelector('.research-stream-tail').textContent = ev.snippet || '';
}

function removeStream() {
  document.getElementById('research-stream')?.remove();
}

async function showDetail(id) {
  stopEvents();
  setBackBtn(true);
  let job;
  try {
    job = await research.get(id);
  } catch {
    location.hash = '';
    return;
  }

  const awaitingReddit = job.status === 'awaiting_reddit';
  const running = job.status === 'running' || job.status === 'pending';
  const displayTitle = job.title || job.query;
  const promptHtml = job.title && job.query
    ? `<p class="research-detail-prompt">${escapeHtml(job.query)}</p>`
    : '';

  main.innerHTML = `
    <div class="research-detail-heading">
      <div class="research-detail-copy">
        <h1 class="research-detail-query">${escapeHtml(displayTitle)}</h1>
        ${promptHtml}
      </div>
      <div class="research-detail-actions">
        ${job.final_report ? `
          <button id="research-remix" class="btn btn-sm btn-secondary">remix</button>
          <button id="research-dl-md" class="btn btn-sm btn-secondary">Markdown</button>
          <button id="research-dl-html" class="btn btn-sm btn-secondary">HTML</button>` : ''}
        ${running || awaitingReddit
          ? '<button id="research-cancel" class="btn btn-sm btn-secondary">cancel</button>'
          : '<button id="research-delete" class="btn btn-sm btn-danger">delete</button>'}
      </div>
    </div>
    <div class="research-detail-meta">
      ${statusBadge(job.status)}
      ${job.mode === 'brainstorm' ? `<span>brainstorm${job.force_search ? ' · always searches' : ''}</span>` : ''}
      ${job.deep_report ? '<span>in-depth</span>' : ''}
      ${job.pause_reddit_import ? '<span>Reddit import</span>' : ''}
      <span>${escapeHtml(job.model)}</span>
      ${job.effort ? `<span>effort: ${EFFORT_NAMES[job.effort] || job.effort}</span>` : ''}
      <span>${formatDate(job.created_at)}</span>
      ${job.elapsed_ms ? `<span>${formatDuration(job.elapsed_ms)}</span>` : ''}
    </div>
    ${running ? '<div id="research-progress" class="research-progress"></div>' : ''}
    ${awaitingReddit ? redditWaitingPanel(job) : ''}
    ${job.final_report ? remixShelf(job) : ''}
    <div id="research-error"></div>
    <div id="research-report" class="research-report"></div>`;

  document.getElementById('research-cancel')?.addEventListener('click', async () => {
    try { await research.cancel(id); showDetail(id); } catch { /* already finished */ }
  });
  document.getElementById('research-delete')?.addEventListener('click', async () => {
    if (!confirm('Delete this research?')) return;
    await research.delete(id);
    location.hash = '';
  });
  document.getElementById('research-dl-md')?.addEventListener('click', () => {
    const heading = job.title || job.query;
    const preamble = job.title && job.query ? `# ${job.title}\n\n${job.query}\n\n` : `# ${heading}\n\n`;
    downloadFile(`${slugify(heading)}.md`, 'text/markdown', preamble + job.final_report);
  });
  document.getElementById('research-remix')?.addEventListener('click', () => {
    location.hash = `${id}/remix/new`;
  });
  main.querySelectorAll('.research-remix-item').forEach((el) => {
    el.addEventListener('click', () => { location.hash = `${id}/remix/${el.dataset.remixId}`; });
  });
  document.getElementById('research-dl-html')?.addEventListener('click', () => {
    const heading = job.title || job.query;
    const headingHtml = job.title && job.query
      ? `<h1>${escapeHtml(job.title)}</h1>\n<p>${escapeHtml(job.query)}</p>\n`
      : `<h1>${escapeHtml(heading)}</h1>\n`;
    downloadFile(`${slugify(heading)}.html`, 'text/html',
      htmlDocument(heading, headingHtml + render(job.final_report)));
  });

  if (job.status === 'error' && job.error) {
    document.getElementById('research-error').innerHTML =
      `<div class="research-error">${escapeHtml(job.error)}</div>`;
  }

  if (job.final_report) {
    document.getElementById('research-report').innerHTML = render(job.final_report);
  } else if (!running && job.report) {
    // Failed or cancelled part-way through — show the evolving report that was saved.
    document.getElementById('research-report').innerHTML =
      '<p class="research-section-label">partial report</p>' + render(job.report);
  }

  if (running) watchProgress(id);
  if (awaitingReddit) wireRedditWaitingPanel(id, job);
}

function remixShelf(job) {
  if (!job.remixes?.length) return '';
  const items = (job.remixes || []).map((remix, index) => {
    const label = remix.direction || `Remix ${(job.remixes.length - index)}`;
    return `<button class="research-remix-item" data-remix-id="${remix.id}">
      <span>${escapeHtml(label)}</span>
      <small>${escapeHtml(remix.model)} · ${formatDate(remix.created_at)}</small>
    </button>`;
  }).join('');
  return `<section class="research-remixes">
    <p class="research-section-label">remixes</p>
    <div class="research-remix-list">${items}</div>
  </section>`;
}

function setReportBackBtn(jobID) {
  const btn = document.getElementById('back-to-menu');
  btn.href = `#${jobID}`;
  btn.onclick = null;
  document.getElementById('back-label').textContent = 'report';
}

async function showRemixForm(jobID) {
  stopEvents();
  setReportBackBtn(jobID);
  let job;
  try { job = await research.get(jobID); } catch { location.hash = ''; return; }
  if (!job.final_report) { location.hash = jobID; return; }
  const options = ['<option value="">default model</option>']
    .concat(modelList.map((m) => `<option value="${escapeHtml(m.name)}">${escapeHtml(m.display_name || m.name)}</option>`))
    .join('');
  main.innerHTML = `<section class="card remix-form">
    <p class="eyebrow">Report remix</p>
    <h1>Make this easier to explore</h1>
    <p class="remix-intro">The model will redesign <strong>${escapeHtml(job.title || job.query)}</strong> as a self-contained HTML document, using diagrams and visual structure where they help.</p>
    <label class="remix-field">Model<select id="remix-model" class="input">${options}</select></label>
    <label class="remix-field">Optional direction<textarea id="remix-direction" class="input textarea" rows="5" maxlength="2000" placeholder="For example: use earthy greens and browns, with simple typography"></textarea></label>
    <div id="remix-progress" class="remix-progress" hidden aria-live="polite">
      <div class="remix-progress-heading"><span id="remix-progress-label">Preparing document</span><span id="remix-progress-count"></span></div>
      <div id="remix-progress-preview" class="remix-progress-preview"></div>
    </div>
    <div id="remix-error"></div>
    <div class="remix-form-actions">
      <a class="btn btn-secondary" href="#${jobID}">cancel</a>
      <button id="remix-go" class="btn btn-primary">create remix</button>
    </div>
  </section>`;
  document.getElementById('remix-go').addEventListener('click', () => {
    const btn = document.getElementById('remix-go');
    const error = document.getElementById('remix-error');
    const model = document.getElementById('remix-model');
    const direction = document.getElementById('remix-direction');
    const progress = document.getElementById('remix-progress');
    const progressLabel = document.getElementById('remix-progress-label');
    const progressCount = document.getElementById('remix-progress-count');
    const progressPreview = document.getElementById('remix-progress-preview');
    btn.setAttribute('disabled', '');
    btn.setAttribute('aria-busy', 'true');
    model.disabled = true;
    direction.disabled = true;
    btn.textContent = 'designing document…';
    error.innerHTML = '';
    progress.hidden = false;
    let completed = false;
    const reset = (err) => {
      error.innerHTML = `<div class="research-error">${escapeHtml(err.message)}</div>`;
      btn.removeAttribute('disabled');
      btn.removeAttribute('aria-busy');
      model.disabled = false;
      direction.disabled = false;
      btn.textContent = 'create remix';
    };
    research.createRemix(jobID, model.value, direction.value.trim(), {
      onEvent: (ev) => {
        if (ev.error) { reset(new Error(ev.error)); return; }
        if (ev.phase === 'connecting') progressLabel.textContent = 'Connecting to model';
        if (ev.phase === 'generating') {
          progressLabel.textContent = 'Designing document';
          progressCount.textContent = `${ev.generated.toLocaleString()} characters`;
          progressPreview.textContent = ev.snippet || '';
        }
        if (ev.phase === 'validating') progressLabel.textContent = 'Checking document';
        if (ev.phase === 'saving') progressLabel.textContent = 'Saving remix';
        if (ev.phase === 'complete' && ev.remix) {
          completed = true;
          location.hash = `${jobID}/remix/${ev.remix.id}`;
        }
      },
      onDone: () => {
        if (!completed && btn.disabled && !error.textContent) reset(new Error('The remix stream ended before the document was saved.'));
      },
      onError: reset,
    });
  });
}

async function showRemix(jobID, remixID) {
  stopEvents();
  setReportBackBtn(jobID);
  let remix;
  try { remix = await research.getRemix(jobID, remixID); } catch { location.hash = jobID; return; }
  const label = remix.direction || 'Designed report';
  main.innerHTML = `<div class="remix-view-header">
    <div><p class="eyebrow">Report remix</p><h1>${escapeHtml(label)}</h1><p>${escapeHtml(remix.model)} · ${formatDate(remix.created_at)}</p></div>
    <button id="remix-download" class="btn btn-sm btn-secondary">download .html</button>
  </div>
  <iframe class="remix-document" title="${escapeHtml(label)}" sandbox src="/api/research/${jobID}/remixes/${remixID}/document"></iframe>`;
  document.getElementById('remix-download').addEventListener('click', () => {
    downloadFile(`research-remix-${remix.id}.html`, 'text/html', remix.html);
  });
}

function redditWaitingPanel(job) {
  const request = job.reddit_request;
  if (!request) return '<div class="research-error">The persisted Reddit request could not be loaded.</div>';
  const pages = request.pages.map((page) => `<li><a href="${escapeHtml(page.url)}" target="_blank" rel="noopener noreferrer">${escapeHtml(page.title || page.url)}</a></li>`).join('');
  return `
    <section class="card reddit-waiting">
      <p class="eyebrow">Browser import required</p>
      <h2 class="h4">Capture Reddit results</h2>
      <p class="body-sm">Give this request to the Save Reddit extension, then paste or upload its response. Waiting time does not count against the research limit.</p>
      <ol class="reddit-url-list">${pages}</ol>
      <div class="reddit-actions">
        <button id="reddit-copy-request" class="btn btn-secondary">copy request</button>
        <button id="reddit-download-request" class="btn btn-secondary">download request</button>
      </div>
      <label class="research-field">Extension response
        <textarea id="reddit-response" class="input textarea code" rows="10" placeholder="Paste response JSON"></textarea>
      </label>
      <label class="research-field">Or upload response JSON
        <input id="reddit-response-file" class="input" type="file" accept="application/json,.json">
      </label>
      <div class="reddit-actions">
        <button id="reddit-import" class="btn btn-primary">validate and continue</button>
        <button id="reddit-skip" class="btn btn-secondary">skip Reddit sources</button>
      </div>
      <p id="reddit-feedback" class="body-sm" aria-live="polite"></p>
    </section>`;
}

function wireRedditWaitingPanel(id, job) {
  const requestText = JSON.stringify(job.reddit_request, null, 2);
  const feedback = document.getElementById('reddit-feedback');
  document.getElementById('reddit-copy-request').addEventListener('click', async () => {
    try {
      await copyToClipboard(requestText);
      feedback.textContent = 'Request copied.';
    } catch (error) {
      feedback.textContent = error.message;
    }
  });
  document.getElementById('reddit-download-request').addEventListener('click', () => {
    const name = job.reddit_request.name || `reddit-request-${job.reddit_request.request_id}`;
    downloadFile(`${name}.json`, 'application/json', requestText);
  });
  document.getElementById('reddit-response-file').addEventListener('change', async (event) => {
    const [file] = event.target.files;
    if (file) document.getElementById('reddit-response').value = await file.text();
  });
  document.getElementById('reddit-import').addEventListener('click', async () => {
    feedback.textContent = 'Validating response…';
    try {
      const response = JSON.parse(document.getElementById('reddit-response').value);
      await research.importReddit(id, response);
      await showDetail(id);
    } catch (error) {
      feedback.textContent = error instanceof SyntaxError ? 'Response is not valid JSON.' : error.message;
    }
  });
  document.getElementById('reddit-skip').addEventListener('click', async () => {
    if (!confirm('Continue without these Reddit sources?')) return;
    feedback.textContent = 'Skipping Reddit sources…';
    try {
      await research.skipReddit(id, job.reddit_request.request_id);
      await showDetail(id);
    } catch (error) { feedback.textContent = error.message; }
  });
}

function watchProgress(id) {
  const log = document.getElementById('research-progress');
  const append = (text, cls = '') => {
    if (!log) return;
    log.querySelector('.current')?.classList.remove('current');
    const line = document.createElement('div');
    line.className = `progress-line current ${cls}`;
    line.textContent = text;
    log.appendChild(line);
    log.scrollTop = log.scrollHeight;
  };

  abortEvents = research.events(id, {
    onEvent: (ev) => {
      if (ev.status) {
        // Terminal event — reload the job to pick up the final report.
        stopEvents();
        if (location.hash.slice(1) === String(id)) showDetail(id);
        return;
      }
      if (ev.generated) {
        updateStream(ev);
        return;
      }
      removeStream(); // any non-streaming event means the generation finished
      append(progressLine(ev), ev.phase === 'warning' ? 'warn' : '');
    },
    onError: () => append('connection to progress stream lost — reload to reconnect', 'warn'),
  });
}

// ── Routing ─────────────────────────────────────────────────

function route() {
  const path = location.hash.slice(1);
  const match = path.match(/^(\d+)\/remix\/(new|\d+)$/);
  if (match && match[2] === 'new') showRemixForm(match[1]);
  else if (match) showRemix(match[1], match[2]);
  else if (path) showDetail(path);
  else showList();
}

async function init() {
  if (!await requireAuth()) return;
  try {
    modelList = await models.list();
  } catch {
    modelList = [];
  }
  try {
    formDefaults = await research.defaults();
  } catch {
    // keep the built-in fallback defaults
  }
  window.addEventListener('hashchange', route);
  route();
}

init();
