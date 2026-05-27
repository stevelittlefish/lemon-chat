const handlers = {};
let socket = null;

export function on(type, fn) {
  handlers[type] = fn;
}

export function connect() {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  socket = new WebSocket(`${proto}//${location.host}/ws`);

  socket.addEventListener('message', (e) => {
    let event;
    try { event = JSON.parse(e.data); } catch { return; }
    handlers[event.type]?.(event);
  });

  socket.addEventListener('close', () => {
    socket = null;
    setTimeout(connect, 3000);
  });

  socket.addEventListener('error', () => {
    socket?.close();
  });
}
