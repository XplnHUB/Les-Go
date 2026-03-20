

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);

    if (url.pathname === "/" || url.pathname === "/index.html") {
      return new Response(html, { headers: { "Content-Type": "text/html" } });
    }
    if (url.pathname === "/app.js") {
      const jsContent = js.replace("WS_HOST", url.host);
      return new Response(jsContent, { headers: { "Content-Type": "application/javascript" } });
    }
    if (url.pathname === "/health") {
      return new Response(JSON.stringify({ status: "ok" }), { headers: { "Content-Type": "application/json" } });
    }

    if (request.headers.get("Upgrade") === "websocket") {
      const id = env.RELAY.idFromName("global-relay");
      const obj = env.RELAY.get(id);
      return obj.fetch(request);
    }

    return new Response("Not Found", { status: 404 });
  },
};

export class RelayServer {
  constructor(state, env) {
    this.sessions = new Map();
  }

  async fetch(request) {
    const pair = new WebSocketPair();
    const [client, server] = Object.values(pair);

    server.accept();
    let myID = null;

    server.addEventListener("message", (msg) => {
      try {
        const packet = JSON.parse(msg.data);

        if (packet.type === "register") {
          myID = packet.from;
          this.sessions.set(myID, server);
          console.log(`Registered: ${myID}`);
          return;
        }

        if (packet.type === "heartbeat") return;

        const target = this.sessions.get(packet.to);
        if (target) {
          target.send(JSON.stringify(packet));
        }
      } catch (err) {
        console.error("Relay error:", err);
      }
    });

    server.addEventListener("close", () => {
      if (myID) {
        this.sessions.delete(myID);
      }
    });

    return new Response(null, { status: 101, webSocket: client });
  }
}

const html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Les'Go - Universal Web Chat</title>
    <style>
        :root { --primary: #00d2ff; --secondary: #3a7bd5; --bg: #0f172a; --text: #f8fafc; }
        body { background: var(--bg); color: var(--text); font-family: system-ui, -apple-system, sans-serif; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
        .container { width: 95%; max-width: 900px; height: 85vh; display: flex; background: rgba(255,255,255,0.05); backdrop-filter: blur(20px); border-radius: 24px; border: 1px solid rgba(255,255,255,0.1); overflow: hidden; box-shadow: 0 20px 50px rgba(0,0,0,0.3); }
        .sidebar { width: 280px; border-right: 1px solid rgba(255,255,255,0.1); display: flex; flex-direction: column; padding: 24px; background: rgba(0,0,0,0.2); }
        .logo { font-size: 28px; font-weight: 900; background: linear-gradient(45deg, var(--primary), var(--secondary)); -webkit-background-clip: text; -webkit-text-fill-color: transparent; margin-bottom: 24px; }
        .my-id-container { background: rgba(255,255,255,0.05); padding: 16px; border-radius: 12px; border: 1px solid rgba(255,255,255,0.1); margin-bottom: 24px; }
        .chat-area { flex: 1; display: flex; flex-direction: column; background: rgba(255,255,255,0.02); }
        .messages { flex: 1; padding: 24px; overflow-y: auto; display: flex; flex-direction: column; gap: 12px; }
        .message { padding: 12px 16px; border-radius: 16px; max-width: 75%; font-size: 14px; line-height: 1.4; color: white; }
        .message.sent { align-self: flex-end; background: linear-gradient(135deg, var(--secondary), #00d2ff); border-bottom-right-radius: 4px; }
        .message.received { align-self: flex-start; background: rgba(255,255,255,0.1); border-bottom-left-radius: 4px; }
        .chat-input-container { padding: 20px; border-top: 1px solid rgba(255,255,255,0.1); display: flex; gap: 12px; }
        input { flex: 1; background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.1); color: white; padding: 12px 16px; border-radius: 12px; outline: none; transition: 0.2s; }
        input:focus { border-color: var(--primary); background: rgba(255,255,255,0.08); }
        button { background: linear-gradient(45deg, var(--primary), var(--secondary)); border: none; color: white; padding: 12px 24px; border-radius: 12px; cursor: pointer; font-weight: 700; transition: 0.2s; }
        button:hover { transform: translateY(-2px); filter: brightness(1.1); }
        .hidden { display: none !important; }
        .overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: var(--bg); z-index: 100; display: flex; align-items: center; justify-content: center; transition: 0.5s; }
        .overlay.fade-out { opacity: 0; pointer-events: none; }
    </style>
</head>
<body>
    <div id="setupOverlay" class="overlay"><div style="text-align: center;"><h1 style="color:var(--primary); font-size:40px; margin-bottom:10px;">Les'Go</h1><p id="status">Securing your session...</p></div></div>
    <div class="container">
        <aside class="sidebar">
            <div class="logo">Les'Go</div>
            <div class="my-id-container">
                <div style="font-size: 10px; opacity: 0.5; margin-bottom: 4px; letter-spacing: 1px;">MY DEVICE ID</div>
                <div id="myID" style="font-family: monospace; font-size: 20px; color: var(--primary); font-weight: bold;">--------</div>
            </div>
            <div style="margin-top: auto;">
                <div style="font-size: 10px; opacity: 0.5; margin-bottom: 8px; letter-spacing: 1px;">START CHAT</div>
                <input type="text" id="targetIDInput" placeholder="Enter Peer ID" maxlength="10" style="width: 100%; margin-bottom: 12px; box-sizing: border-box;">
                <button id="connectBtn" style="width: 100%;">Connect</button>
            </div>
        </aside>
        <main class="chat-area">
            <div id="emptyState" class="messages" style="justify-content: center; align-items: center; opacity: 0.3; text-align: center;">
                <div><p style="font-size: 40px; margin: 0;">🔒</p><p>End-to-End Encrypted Communication</p></div>
            </div>
            <div id="chatInterface" class="hidden" style="display: flex; flex-direction: column; flex: 1;">
                <div style="padding: 24px; border-bottom: 1px solid rgba(255,255,255,0.1); background: rgba(0,0,0,0.1);"><h2 id="currentPeerID" style="font-size: 18px; margin: 0;">Connected</h2></div>
                <div id="messageList" class="messages"></div>
                <form id="chatForm" class="chat-input-container">
                    <input type="text" id="messageInput" placeholder="Type an encrypted message..." autocomplete="off">
                    <button type="submit">Send</button>
                </form>
            </div>
        </main>
    </div>
    <script src="app.js"></script>
</body>
</html>`;

const js = `
const PACKET_TYPES = { REGISTER: "register", REQUEST: "connect_request", ACCEPT: "connect_accept", PUBKEY: "public_key", AES: "aes_key", MSG: "message", HB: "heartbeat", DISC: "disconnect" };
class LesGoClient {
    constructor() {
        this.ws = null; this.myID = ""; this.keyPair = null; this.sessions = new Map(); this.activePeer = null;
        this.els = { id: document.getElementById('myID'), target: document.getElementById('targetIDInput'), btn: document.getElementById('connectBtn'), chat: document.getElementById('chatInterface'), empty: document.getElementById('emptyState'), peer: document.getElementById('currentPeerID'), list: document.getElementById('messageList'), input: document.getElementById('messageInput'), form: document.getElementById('chatForm'), overlay: document.getElementById('setupOverlay') };
        this.init();
    }
    async init() {
        this.myID = Array.from({length:10}, () => Math.floor(Math.random()*10)).join('');
        this.els.id.innerText = this.myID;
        this.keyPair = await crypto.subtle.generateKey({name: "RSA-OAEP", modulusLength: 2048, publicExponent: new Uint8Array([1,0,1]), hash: "SHA-256"}, true, ["encrypt", "decrypt"]);
        this.connect();
    }
    connect() {
        this.ws = new WebSocket("wss://WS_HOST");
        this.ws.onopen = () => { this.send(PACKET_TYPES.REGISTER, "", ""); this.els.overlay.classList.add('fade-out'); setInterval(() => this.send(PACKET_TYPES.HB, "", ""), 30000); };
        this.ws.onmessage = async (e) => { const p = JSON.parse(e.data); await this.handle(p); };
        this.els.btn.onclick = () => { const t = this.els.target.value; if(t.length===10) this.send(PACKET_TYPES.REQUEST, t, ""); };
        this.els.form.onsubmit = async (e) => { e.preventDefault(); const m = this.els.input.value; if(m && this.activePeer) { await this.sendMessage(this.activePeer, m); this.els.input.value = ""; } };
    }
    async handle(p) {
        switch(p.type) {
            case PACKET_TYPES.REQUEST: if(confirm("Request from "+p.from)) this.send(PACKET_TYPES.ACCEPT, p.from, ""); break;
            case PACKET_TYPES.ACCEPT: const pub = await this.exportPub(); this.send(PACKET_TYPES.PUBKEY, p.from, pub); break;
            case PACKET_TYPES.PUBKEY: 
                const peerPk = await this.importPub(p.payload);
                if(!this.sessions.has(p.from)) {
                    const aes = await crypto.subtle.generateKey({name: "AES-GCM", length: 256}, true, ["encrypt", "decrypt"]);
                    this.sessions.set(p.from, {aes, msgs: []});
                    this.send(PACKET_TYPES.PUBKEY, p.from, await this.exportPub());
                    const rawAes = await crypto.subtle.exportKey("raw", aes);
                    const encAes = await crypto.subtle.encrypt({name: "RSA-OAEP"}, peerPk, rawAes);
                    this.send(PACKET_TYPES.AES, p.from, btoa(String.fromCharCode(...new Uint8Array(encAes))));
                    this.open(p.from);
                }
                break;
            case PACKET_TYPES.AES:
                const decAes = await crypto.subtle.decrypt({name: "RSA-OAEP"}, this.keyPair.privateKey, new Uint8Array(atob(p.payload).split('').map(c=>c.charCodeAt(0))));
                const aes = await crypto.subtle.importKey("raw", decAes, "AES-GCM", true, ["encrypt", "decrypt"]);
                this.sessions.set(p.from, {aes, msgs: []});
                this.open(p.from);
                break;
            case PACKET_TYPES.MSG:
                const s = this.sessions.get(p.from);
                if(s) { const txt = await this.decrypt(s.aes, p.payload); this.addMsg(p.from, txt, 'received'); }
                break;
        }
    }
    async sendMessage(t, text) {
        const s = this.sessions.get(t);
        const enc = await this.encrypt(s.aes, text);
        this.send(PACKET_TYPES.MSG, t, enc);
        this.addMsg(t, text, 'sent');
    }
    send(type, to, payload) { this.ws.send(JSON.stringify({type, from: this.myID, to, payload, timestamp: Date.now()})); }
    open(id) { this.activePeer = id; this.els.peer.innerText = "Chat with " + id; this.els.empty.classList.add('hidden'); this.els.chat.classList.remove('hidden'); }
    addMsg(id, text, type) { if(this.activePeer === id) { const d = document.createElement('div'); d.className = 'message '+type; d.innerText = text; this.els.list.appendChild(d); this.els.list.scrollTop = this.els.list.scrollHeight; } }
    async exportPub() { const e = await crypto.subtle.exportKey("spki", this.keyPair.publicKey); return btoa(String.fromCharCode(...new Uint8Array(e))); }
    async importPub(b) { return await crypto.subtle.importKey("spki", new Uint8Array(atob(b).split('').map(c=>c.charCodeAt(0))), {name: "RSA-OAEP", hash: "SHA-256"}, true, ["encrypt"]); }
    async encrypt(k, t) {
        const iv = crypto.getRandomValues(new Uint8Array(12));
        const enc = await crypto.subtle.encrypt({name: "AES-GCM", iv}, k, new TextEncoder().encode(t));
        const c = new Uint8Array(12 + enc.byteLength); c.set(iv); c.set(new Uint8Array(enc), 12);
        return btoa(String.fromCharCode(...c));
    }
    async decrypt(k, b) {
        const c = new Uint8Array(atob(b).split('').map(x=>x.charCodeAt(0)));
        const dec = await crypto.subtle.decrypt({name: "AES-GCM", iv: c.slice(0,12)}, k, c.slice(12));
        return new TextDecoder().decode(dec);
    }
}
window.onload = () => new LesGoClient();
`;
