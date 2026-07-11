# User-assisted Reddit imports for research

## Purpose

Reddit search results are useful research sources, but lemon-chat cannot
reliably fetch their content directly. This feature lets a user who is already
logged into Reddit collect selected search-result threads in their browser and
return them to a paused research job.

The mode is optional and disabled by default. It does not add Reddit
credentials, proxy rotation, CAPTCHA handling, or undocumented Reddit API calls
to lemon-chat.

## User flow

1. The user starts a research job with **Pause to import Reddit results**
   enabled.
2. A search round runs normally and divides new results into ordinary URLs and
   Reddit thread URLs.
3. If the round contains Reddit URLs, the engine checkpoints the complete
   pending round and changes the job status to `awaiting_reddit`. No research
   goroutine remains blocked, and waiting time does not consume the job's time
   budget.
4. The research panel presents a versioned request bundle containing a stable
   request ID and the canonical Reddit URLs.
5. The user gives the bundle to the Save Reddit browser extension. The
   extension visits the URLs sequentially using the user's normal browser
   session, extracts rendered posts and comments, and produces a response
   bundle with explicit completeness warnings.
6. The user pastes or uploads the response in the research panel. Alternatively,
   they can skip the Reddit sources and continue.
7. The server validates the bundle, converts each page to normalized text, and
   passes it through the same goal-based, prompt-injection-guarded extraction
   path as other webpages.
8. The engine processes the pending ordinary URLs, merges all successful
   findings, synthesizes the round, checkpoints it, and continues.

## Architecture

### Durable pause and resume

`awaiting_reddit` is a persisted job status, not a long-running handler or a
goroutine waiting on a channel. Startup recovery must resume only `pending` and
`running` jobs. An accepted import or explicit skip atomically changes the job
back to a resumable state and starts a new run from its checkpoint.

The pending checkpoint must preserve enough information to complete exactly the
same round without repeating its LLM query-generation or web-search calls:

- round number and creative-round level;
- generated queries;
- ordinary URLs awaiting extraction;
- canonical Reddit URLs awaiting import;
- stable, unpredictable request ID;
- any accepted import response;
- elapsed active research time at the pause.

Submission and resume must be idempotent. A repeated response for an already
accepted request must not create duplicate findings or start two runners.

### Interchange contract

The browser boundary uses versioned JSON. JSON is retained until validation and
normalization are complete; Markdown is generated only as an LLM-facing and
diagnostic representation.

A request contains:

- schema version;
- research job/request identifiers;
- canonical URL and search-result title for each requested thread;
- capture limits understood by the extension.

A response contains one result per requested URL:

- requested and canonical URLs;
- post title, subreddit, author, body, permalink, and optional score;
- flattened comments with stable order, depth, author, body, permalink, and
  optional score;
- capture timestamp;
- warnings, failure reason, and an explicit completeness indicator.

The server accepts content only for URLs in the pending request. It enforces
per-field, per-page, comment-count, nesting-depth, and total-body limits. It
deduplicates pages and comments, rejects unknown schema versions, and never logs
imported bodies.

One captured thread normally becomes one `Finding`. Raw comments are not
treated as independent sources.

### URL handling

Canonicalization recognizes `reddit.com`, `www.reddit.com`, `old.reddit.com`,
`new.reddit.com`, and resolvable `redd.it` thread links. Tracking/query
parameters and fragments are removed. Post and comment permalinks retain the
thread identity so multiple results from one thread can be grouped without
discarding a specifically matched comment permalink.

Already pending, imported, skipped, or analyzed thread identities do not cause
another pause in a later round.

### Browser extension

The first extension targets Chromium Manifest V3 and requests host access only
for Reddit. It does not read or export cookies and does not call Reddit's
undocumented APIs. It operates on rendered pages using the user's existing
logged-in browser session.

Threads are processed sequentially with configurable limits:

- delay between pages;
- maximum processing time and comments per thread;
- maximum scroll/expand attempts;
- stop, retry, and skip controls.

The extension must report partial capture honestly. Deleted, private,
quarantined, age-gated, rate-limited, or structurally unrecognized pages produce
warnings or failures rather than silently empty documents. Reddit DOM changes
will require selector maintenance, and complete comment-tree capture cannot be
guaranteed.

## Development harness

A debug-only single-pass page proves the browser/server boundary without
running a multi-round research job. It is not linked in normal navigation and
returns 404 unless debug mode is enabled.

The page can:

1. run one SearXNG query scoped with `site:reddit.com` or accept manual URLs;
2. show canonicalization, grouping, and rejection results;
3. export an extension request bundle;
4. paste or upload and validate a response bundle;
5. show structured captured content, warnings, counts, and normalized text;
6. optionally run the production goal-based extractor once;
7. render the resulting `Finding` with research formatting.

Validation-only operation must not require an LLM call. Synthetic fixtures are
used for automated tests; captured Reddit discussions are not committed to the
repository.

## Security and privacy

- All research and import endpoints require authentication and job ownership.
- Request IDs are checked in addition to job IDs.
- Imported text is escaped for UI display and wrapped by the existing untrusted
  context guard before an LLM sees it.
- Raw import bodies are not included in normal logs.
- Server and browser enforce conservative size and work limits.
- The extension sends data only through explicit user export; a future direct
  localhost handoff would require a separate security review.
- Source permalinks and capture warnings remain attached to findings.

## Delivery sequence

1. Shared contract, URL handling, normalization, fixtures, and unit tests.
2. Debug-only single-pass harness.
3. Browser extension and manual end-to-end testing against the harness.
4. Per-job option and durable database state.
5. Engine pause/resume plus import/skip endpoints.
6. Research-panel workflow and full recovery/end-to-end tests.

Each step should be committed independently after its relevant tests pass.
