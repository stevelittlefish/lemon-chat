import { requireAuth } from './settings-auth.js';

const $ = (id) => document.getElementById(id);
let currentRequest = null;

async function post(payload) {
  const response = await fetch('/api/debug/reddit-import', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Requested-With': 'XMLHttpRequest',
    },
    body: JSON.stringify(payload),
  });
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`);
  return data;
}

function parseJSON(id, label) {
  try { return JSON.parse($(id).value); }
  catch { throw new Error(`${label} is not valid JSON`); }
}

$('prepare').addEventListener('click', async () => {
  $('prepare-output').textContent = 'Preparing…';
  try {
    const urls = $('urls').value.split('\n').map((url) => url.trim()).filter(Boolean).map((url) => ({ url }));
    const data = await post({ action: 'prepare', query: $('query').value.trim(), urls });
    currentRequest = data.request || null;
    $('request').value = currentRequest ? JSON.stringify(currentRequest, null, 2) : '';
    $('prepare-output').textContent = JSON.stringify(data, null, 2);
  } catch (error) { $('prepare-output').textContent = error.message; }
});

async function run(action) {
  $('result').textContent = action === 'extract' ? 'Extracting…' : 'Validating…';
  try {
    const data = await post({
      action,
      request: parseJSON('request', 'Request bundle'),
      response: parseJSON('response', 'Response bundle'),
      goal: $('goal').value.trim(),
      model: $('model').value.trim(),
    });
    $('result').textContent = JSON.stringify(data, null, 2);
  } catch (error) { $('result').textContent = error.message; }
}

$('validate').addEventListener('click', () => run('validate'));
$('extract').addEventListener('click', () => run('extract'));
$('response-file').addEventListener('change', async (event) => {
  const [file] = event.target.files;
  if (file) $('response').value = await file.text();
});
$('copy-request').addEventListener('click', async () => navigator.clipboard.writeText($('request').value));
$('download-request').addEventListener('click', () => {
  const blob = new Blob([$('request').value], { type: 'application/json' });
  const link = document.createElement('a');
  link.href = URL.createObjectURL(blob);
  link.download = `reddit-import-${currentRequest?.request_id || 'request'}.json`;
  link.click();
  URL.revokeObjectURL(link.href);
});

requireAuth();
