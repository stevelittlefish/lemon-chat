// Research page: start research jobs, watch live progress, read reports.
import { requireAuth } from './settings-auth.js';
import { research, models } from './api.js';
import { render } from './markdown.js';
import { escapeHtml } from './utils.js';

const main = document.getElementById('research-main');

let modelList = [];
let abortEvents = null; // aborts the active SSE subscription

function stopEvents() {
  if (abortEvents) { abortEvents(); abortEvents = null; }
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
    default: return `<span class="badge">${escapeHtml(status)}</span>`;
  }
}

// ── List view ───────────────────────────────────────────────

async function showList() {
  stopEvents();
  const jobs = await research.list();

  const options = ['<option value="">default model</option>']
    .concat(modelList.map((m) =>
      `<option value="${escapeHtml(m.name)}">${escapeHtml(m.display_name || m.name)}</option>`))
    .join('');

  const items = jobs.map((j) => `
    <div class="card card-interactive research-item" data-id="${j.id}">
      <span class="research-item-query">${escapeHtml(j.query)}</span>
      <span class="research-item-meta">${formatDate(j.created_at)}</span>
      ${statusBadge(j.status)}
    </div>`).join('');

  main.innerHTML = `
    <div class="card research-form">
      <h2 class="research-form-title">new research</h2>
      <textarea id="research-query" class="input textarea" rows="3"
        placeholder="what do you want to know? the more specific the question, the better the report"></textarea>
      <div class="research-form-row">
        <select id="research-model" class="input">${options}</select>
        <button id="research-start" class="btn btn-primary">start research</button>
      </div>
    </div>
    <p class="research-section-label">past research</p>
    <div class="research-list">
      ${items || '<p class="research-empty">no research yet — ask a question above to get started</p>'}
    </div>`;

  document.getElementById('research-start').addEventListener('click', startResearch);
  document.getElementById('research-query').addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) startResearch();
  });
  main.querySelectorAll('.research-item').forEach((el) => {
    el.addEventListener('click', () => { location.hash = el.dataset.id; });
  });
}

async function startResearch() {
  const query = document.getElementById('research-query').value.trim();
  if (!query) return;
  const model = document.getElementById('research-model').value;
  const btn = document.getElementById('research-start');
  btn.disabled = true;
  try {
    const job = await research.start(query, model);
    location.hash = job.id;
  } catch (err) {
    btn.disabled = false;
    alert(err.message);
  }
}

// ── Detail view ─────────────────────────────────────────────

function progressLine(ev) {
  const host = (url) => { try { return new URL(url).hostname; } catch { return url; } };
  switch (ev.phase) {
    case 'planning': return 'planning research approach';
    case 'searching': return `round ${ev.round} — searching: ${(ev.queries || []).join(' · ')}`;
    case 'reading': return `round ${ev.round} — reading ${host(ev.url)}`;
    case 'analyzing': return `round ${ev.round} — synthesizing (${ev.total_findings} finding${ev.total_findings === 1 ? '' : 's'})`;
    case 'writing': return 'writing final report';
    case 'warning': return ev.message;
    default: return ev.message || ev.phase;
  }
}

async function showDetail(id) {
  stopEvents();
  let job;
  try {
    job = await research.get(id);
  } catch {
    location.hash = '';
    return;
  }

  const running = job.status === 'running' || job.status === 'pending';

  main.innerHTML = `
    <h1 class="research-detail-query">${escapeHtml(job.query)}</h1>
    <div class="research-detail-meta">
      ${statusBadge(job.status)}
      <span>${escapeHtml(job.model)}</span>
      <span>${formatDate(job.created_at)}</span>
      ${job.elapsed_ms ? `<span>${formatDuration(job.elapsed_ms)}</span>` : ''}
      <span class="research-detail-actions">
        ${running
          ? '<button id="research-cancel" class="btn btn-sm btn-secondary">cancel</button>'
          : '<button id="research-delete" class="btn btn-sm btn-danger">delete</button>'}
      </span>
    </div>
    ${running ? '<div id="research-progress" class="research-progress"></div>' : ''}
    <div id="research-error"></div>
    <div id="research-report" class="research-report"></div>`;

  document.getElementById('research-cancel')?.addEventListener('click', async () => {
    try { await research.cancel(id); } catch { /* already finished */ }
  });
  document.getElementById('research-delete')?.addEventListener('click', async () => {
    if (!confirm('Delete this research?')) return;
    await research.delete(id);
    location.hash = '';
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
      append(progressLine(ev), ev.phase === 'warning' ? 'warn' : '');
    },
    onError: () => append('connection to progress stream lost — reload to reconnect', 'warn'),
  });
}

// ── Routing ─────────────────────────────────────────────────

function route() {
  const id = location.hash.slice(1);
  if (id) showDetail(id);
  else showList();
}

async function init() {
  if (!await requireAuth()) return;
  try {
    modelList = await models.list();
  } catch {
    modelList = [];
  }
  window.addEventListener('hashchange', route);
  route();
}

init();
