// Icon.jsx — tiny inline Lucide-style icons. Keeps the chat kit self-contained
// without needing the Lucide runtime script.
// Pass `name` (string) + optional `size` (default 18) + `className`/`style`.

const ICONS = {
  plus:    "M12 5v14M5 12h14",
  search:  "M21 21l-4.35-4.35 M16.5 11a5.5 5.5 0 1 1-11 0 5.5 5.5 0 0 1 11 0",
  send:    "M22 2L11 13 M22 2l-7 20-4-9-9-4 20-7",
  message: "M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z",
  settings:"M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h0a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51h0a1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v0a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z",
  trash:   "M3 6h18 M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6 M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2",
  edit:    "M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7 M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z",
  more:    "M12 13a1 1 0 1 0 0-2 1 1 0 0 0 0 2z M19 13a1 1 0 1 0 0-2 1 1 0 0 0 0 2z M5 13a1 1 0 1 0 0-2 1 1 0 0 0 0 2z",
  check:   "M20 6L9 17l-5-5",
  chevDown:"M6 9l6 6 6-6",
  chevUp:  "M18 15l-6-6-6 6",
  copy:    "M9 9h13v13H9z M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1",
  refresh: "M3 12a9 9 0 0 1 15.5-6.4L21 8 M21 3v5h-5 M21 12a9 9 0 0 1-15.5 6.4L3 16 M3 21v-5h5",
  thumbsUp: "M7 22V11 M2 13v7a2 2 0 0 0 2 2h13.5a2 2 0 0 0 2-1.7l1.4-9a2 2 0 0 0-2-2.3H14V4a2 2 0 0 0-2-2L7 11",
  download:"M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4 M7 10l5 5 5-5 M12 15V3",
  attach:  "M21.44 11.05L12.25 20.24a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48",
  sparkle: "M12 3l1.6 6.4L20 11l-6.4 1.6L12 19l-1.6-6.4L4 11l6.4-1.6z",
  panel:   "M3 3h7v18H3z M14 3h7v18h-7z",
  cmd:     "M18 3a3 3 0 0 0-3 3v12a3 3 0 0 0 3 3 3 3 0 0 0 3-3 3 3 0 0 0-3-3H6a3 3 0 0 0-3 3 3 3 0 0 0 3 3 3 3 0 0 0 3-3V6a3 3 0 0 0-3-3 3 3 0 0 0-3 3 3 3 0 0 0 3 3h12a3 3 0 0 0 3-3 3 3 0 0 0-3-3z",
  info:    "M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20z M12 16v-4 M12 8h.01",
  thumbsDown: "M17 2v11 M22 11v-7a2 2 0 0 0-2-2H6.5a2 2 0 0 0-2 1.7l-1.4 9a2 2 0 0 0 2 2.3H10v3a2 2 0 0 0 2 2l5-9",
  close:   "M18 6L6 18 M6 6l12 12",
  bolt:    "M13 2L3 14h9l-1 8 10-12h-9l1-8z",
  clock:   "M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20z M12 6v6l4 2",
  user:    "M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2 M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8",
  camera:  "M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z M12 17a4 4 0 1 0 0-8 4 4 0 0 0 0 8",
  eye:     "M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7z M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z",
  pencil:  "M12 20h9 M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4z",
  sliders: "M4 21v-7 M4 10V3 M12 21v-9 M12 8V3 M20 21v-5 M20 12V3 M1 14h6 M9 8h6 M17 16h6",
  returnKey: "M9 10L4 15l5 5 M20 4v7a4 4 0 0 1-4 4H4",
  play:    "M6 4l14 8-14 8z",
  stop:    "M6 6h12v12H6z",
  list:    "M8 6h13 M8 12h13 M8 18h13 M3 6h.01 M3 12h.01 M3 18h.01",
  x:       "M18 6L6 18 M6 6l12 12",
};

function Icon({ name, size = 18, className = "", style = {} }) {
  const d = ICONS[name];
  if (!d) return null;
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      style={style}
      aria-hidden="true"
    >
      {d.split(" M").map((part, i) => (
        <path key={i} d={i === 0 ? part : "M" + part} />
      ))}
    </svg>
  );
}

window.Icon = Icon;
