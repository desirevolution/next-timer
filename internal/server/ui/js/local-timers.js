// Local timers deliberately do not sync with NextTimer's Go service or AWTRIX.
// Wall-clock deadlines survive reloads. Browsers cannot guarantee background alarms.
const KEY = 'nexttimer.local-timers.v1';
const TTL = 60_000;
const MAX_DURATION = 7 * 24 * 60 * 60 * 1000;

export function parseTimerDuration(input) {
  const text = String(input || '').trim();
  const parts = [...text.matchAll(/(\d+(?:\.\d+)?)(ms|h|m|s)/g)];
  if (!parts.length || parts.map(p => p[0]).join('') !== text) throw Error('Dauer angeben, z. B. 10m oder 1h30m.');
  const ms = parts.reduce((sum, p) => sum + Number(p[1]) * ({ms:1,h:3600000,m:60000,s:1000}[p[2]]), 0);
  if (!Number.isFinite(ms) || ms < 1000 || ms > MAX_DURATION) throw Error('Dauer muss zwischen 1s und 168h liegen.');
  return Math.round(ms);
}

export class LocalTimers {
  constructor(onChange, onFinish) {
    this.onChange = onChange;
    this.onFinish = onFinish;
    this.timers = this.read();
    window.addEventListener('storage', e => {
      if (e.key !== KEY) return;
      this.timers = this.read();
      this.onChange();
    });
    this.interval = window.setInterval(() => this.tick(), 250);
    window.setTimeout(() => this.tick(), 0);
  }
  read() {
    try {
      const data = JSON.parse(localStorage.getItem(KEY) || '[]');
      if (!Array.isArray(data)) return [];
      return data.filter(t => typeof t.id === 'string' && t.id.startsWith('local-') &&
        ['running','paused','finished'].includes(t.status) && Number.isFinite(t.duration_ms) &&
        Number.isFinite(t.remaining_ms) && Number.isFinite(t.created_at_ms) &&
        Number.isFinite(t.deadline_ms) && typeof t.name === 'string').slice(-100).map(({icon, ...t}) => ({...t,local:true,sound:t.sound!==false,notify:t.notify!==false}));
    } catch { return []; }
  }
  save() {
    try { localStorage.setItem(KEY, JSON.stringify(this.timers)); }
    catch { throw Error('Lokale Timer können nicht gespeichert werden (Browserspeicher voll oder gesperrt).'); }
    this.onChange();
  }
  all() { return this.timers; }
  start(name, duration, options={}) {
    const ms = parseTimerDuration(duration);
    const now = Date.now();
    const t = {id:'local-'+(crypto.randomUUID?.() || `${Date.now()}-${Math.random().toString(36).slice(2)}`),name:(name || '').trim() || 'Timer',duration_ms:ms,
      remaining_ms:ms,deadline_ms:now+ms,created_at_ms:now,finished_at_ms:0,status:'running',local:true,sound:options.sound!==false,notify:options.notify!==false,alarm:options.alarm??null};
    this.timers.push(t);
    this.save();
    return t;
  }
  control(id, action, delta = 0) {
    const t = this.timers.find(x => x.id === id);
    if (!t) throw Error('Lokaler Timer nicht gefunden.');
    const now = Date.now();
    const remain = t.status === 'running' ? Math.max(0,t.deadline_ms-now) : t.remaining_ms;
    if (t.status === 'running' && remain <= 0) this.finish(t, now);
    if (action === 'stop') {this.timers = this.timers.filter(x => x.id !== id);this.save();return;}
    if (action === 'pause' && t.status === 'running') {t.remaining_ms=remain;t.status='paused';}
    else if (action === 'resume' && t.status === 'paused') {t.deadline_ms=now+t.remaining_ms;t.status='running';}
    else if (action === 'adjust' && (t.status === 'running' || t.status === 'paused')) {
      if (remain+delta <= 0) {this.finish(t,now);this.save();return;}
      if (remain+delta > MAX_DURATION) throw Error('Maximale Restzeit: 168h.');
      t.remaining_ms=remain+delta;
      t.duration_ms=Math.max(1000,t.duration_ms+delta);
      if (t.status === 'running') t.deadline_ms=now+t.remaining_ms;
    } else throw Error('Aktion für den aktuellen Zustand nicht möglich.');
    this.save();
  }
  finish(t, now) {
    if(t.status === 'finished') return;
    t.status='finished';t.remaining_ms=0;t.finished_at_ms=now;
    this.onFinish(t);
  }
  tick() {
    const now=Date.now();let changed=false;
    for(const t of this.timers) {
      if(t.status === 'running' && t.deadline_ms <= now) {this.finish(t,now);changed=true;}
    }
    const before=this.timers.length;
    this.timers=this.timers.filter(t=>t.status!=='finished'||now-t.finished_at_ms<TTL);
    if(changed||this.timers.length!==before) this.save();
  }
}
