# Save Reddit for lemon-chat

This unpacked Chromium Manifest V3 extension captures rendered Reddit threads
for the user-assisted research import flow. It uses the browser's existing
Reddit session, processes pages sequentially, and exports JSON only when the
user explicitly clicks **Export response**.

The popup's load-more limit controls how many visible comment expansion links
are clicked on each page and defaults to 5. When the active tab is a supported
Reddit page, **Export current page** captures it without requiring a request
bundle. Batch completion is shown in the popup and with a green **Done** badge
on the extension icon.

## Install for development

1. Open `chrome://extensions` (or the equivalent Chromium extensions page).
2. Enable developer mode.
3. Choose **Load unpacked** and select this `extensions/save-reddit` directory.
4. Pin **Save Reddit for lemon-chat** if desired.

Use the debug harness at `/debug/reddit-import` on a lemon-chat server started
with debug mode enabled to prepare a request and validate the exported response.

The extension requests access only to Reddit hosts. It does not inspect or
export cookies and does not use Reddit APIs. Reddit markup varies and changes;
warnings and failures in the response must be reviewed, and complete comment
capture cannot be guaranteed.
