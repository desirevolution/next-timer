// Browser alarms are best effort: a closed tab/PWA cannot play scheduled audio.
const DEFAULTS_KEY = 'nexttimer.browser-alarm-defaults.v1';
const SEEN_KEY = 'nexttimer.alerted.v1';
const PENDING_KEY = 'nexttimer.pending-browser-alarms.v1';
const TONES = ['none','gong','bell','digital','alert'];
const REPEATS = ['once','three','until_ack'];
const BASE = Object.freeze({tone:'bell',repeat:'once',visual:true,system:true,volume:0.6});
// A repeat is a short audible pattern followed by silence, not one continuous tone.
const THREE_INTERVAL_MS = 3000;
const UNTIL_ACK_INTERVAL_MS = 10000;
const MIN_ALARM_WINDOW_MS = 2000;
let audioContext;
let nextAudioTime = 0;
const active = new Map();
const audioByAlarm = new Map();
let onChange = () => {};
// The tab alarm is purely visual. It never controls the browser alarm itself.
let tabInitialized = false;
let normalTitle = 'NextTimer';
let normalIcon = '';
let iconElement = null;
let tabAlarmVisible = false;
let tabPulse = false;

function updateTabAlarm() {
  if (!tabInitialized) return;
  // Only use the browser tab as a secondary indicator when the page is hidden.
  // A red icon is persistent; title animation is deliberately non-essential
  // because background timers may be throttled or suspended by the browser.
  const pending = [...active.values()].filter(a => a.alarm.visual);
  if (!document.hidden || pending.length === 0) {
    if (tabAlarmVisible) {
      document.title = normalTitle;
      if (iconElement) iconElement.setAttribute('href', normalIcon);
      tabAlarmVisible = false;
    }
    return;
  }
  if (!tabAlarmVisible) {
    if (iconElement) iconElement.setAttribute('href', new URL('../icons/icon-alarm.png', import.meta.url).href);
    tabAlarmVisible = true;
    tabPulse = false;
  }
  const label = pending.length === 1
    ? `${pending[0].name.slice(0, 48)} fertig!`
    : `${pending.length} Timer fertig!`;
  document.title = `${tabPulse ? '◉' : '●'} ${label} – ${normalTitle}`;
}
function toggleTabTitle() {
  if (!tabAlarmVisible) return;
  tabPulse = !tabPulse;
  updateTabAlarm();
}

export const canNotify = () => 'Notification' in window && window.isSecureContext;
export const permission = () => canNotify() ? Notification.permission : 'unsupported';
export async function requestNotifications() {
  if (!canNotify()) throw Error('Systembenachrichtigungen benötigen HTTPS oder localhost und einen unterstützten Browser.');
  return Notification.requestPermission();
}
const clamp = n => Math.min(1,Math.max(0,Number(n)||0));
const valid = a => a && typeof a === 'object' && TONES.includes(a.tone) && REPEATS.includes(a.repeat) &&
  typeof a.visual === 'boolean' && typeof a.system === 'boolean';
export function getDefaults() {
  try {
    const v = JSON.parse(localStorage.getItem(DEFAULTS_KEY)||'null');
    if (valid(v)) return {...v,volume:Number.isFinite(v.volume)?clamp(v.volume):BASE.volume};
  } catch {}
  return {...BASE};
}
export function saveDefaults(options) {
  if (!valid(options)) throw Error('Ungültige Browser-Alarmeinstellungen.');
  const clean = {...options,volume:clamp(options.volume)};
  localStorage.setItem(DEFAULTS_KEY,JSON.stringify(clean));
  return clean;
}
export function resolveAlarm(timer={}) {
  const global = getDefaults();
  const local = valid(timer.alarm) ? timer.alarm : null;
  const resolved = {...global,...local};
  // Old profiles without a volume continue to inherit this browser's level.
  resolved.volume = local && Number.isFinite(local.volume) ? clamp(local.volume) : global.volume;
  // Preserve the intent of pre-profile favorites, which had two booleans.
  if (!local) {
    if (timer.sound === false) resolved.tone='none';
    if (timer.notify === false) {resolved.visual=false;resolved.system=false;}
  }
  return resolved;
}
export function unlockAudio() {
  try {
    audioContext ??= new (window.AudioContext || window.webkitAudioContext)();
    if (audioContext.state === 'suspended') audioContext.resume().catch(()=>{});
  } catch {}
}
function note(type,frequency,start,duration,volume) {
  const osc=audioContext.createOscillator(),gain=audioContext.createGain();
  osc.type=type;
  osc.frequency.setValueAtTime(frequency,start);
  gain.gain.setValueAtTime(0.0001,start);
  gain.gain.exponentialRampToValueAtTime(Math.max(.0002,volume),start+.02);
  gain.gain.exponentialRampToValueAtTime(.0001,start+duration);
  osc.connect(gain);gain.connect(audioContext.destination);
  osc.start(start);osc.stop(start+duration+.02);
  return gain;
}
const patterns = {
  gong:[['sine',440,0,.95,.16],['sine',660,.015,.65,.055],['sine',880,.04,.42,.027]],
  bell:[['sine',880,0,.35,.17],['sine',1320,.035,.28,.07],['sine',1100,.47,.38,.14],['sine',1650,.49,.23,.045]],
  digital:[['square',784,0,.11,.085],['square',784,.20,.11,.085],['square',1046,.40,.20,.085]],
  alert:[['triangle',740,0,.14,.17],['triangle',988,.22,.14,.17],['triangle',740,.44,.14,.17],['triangle',988,.66,.20,.17]]
};
// The visual alert and the audible repeat schedule use the same duration.
export function alarmWindowMs(alarm) {
  if (alarm.repeat === 'until_ack') return Infinity;
  const pattern = patterns[alarm.tone];
  const soundMs = pattern ? Math.max(...pattern.map(([, , offset, duration]) => offset + duration)) * 1000 : 0;
  const lastCycleMs = Math.max(MIN_ALARM_WINDOW_MS, soundMs);
  return (alarm.repeat === 'three' ? 2 * THREE_INTERVAL_MS : 0) + lastCycleMs;
}
export function playFinishSound(tone='bell',volume=0.6,alarmKey=null) {
  if (tone==='none' || !patterns[tone] || volume<=0) return;
  unlockAudio();
  if (!audioContext) return;
  if (audioContext.state === 'suspended') {
    audioContext.resume().then(() => {if(audioContext.state==='running'&&(!alarmKey||active.has(alarmKey)))playFinishSound(tone,volume,alarmKey);}).catch(()=>{});
    return;
  }
  if (audioContext.state !== 'running') return;
  const start=Math.max(audioContext.currentTime+.012,nextAudioTime);
  const items=patterns[tone];
  for (const [wave,freq,offset,duration,level] of items) {
    const gain=note(wave,freq,start+offset,duration,clamp(volume)*level);
    if (alarmKey) {
      if (!audioByAlarm.has(alarmKey)) audioByAlarm.set(alarmKey,new Set());
      audioByAlarm.get(alarmKey).add(gain);
      window.setTimeout(() => {
        audioByAlarm.get(alarmKey)?.delete(gain);
        if (audioByAlarm.get(alarmKey)?.size===0) audioByAlarm.delete(alarmKey);
      },Math.max(0,(start+offset+duration-audioContext.currentTime)*1000)+1000);
    }
  }
  nextAudioTime=start+Math.max(...items.map(i=>i[2]+i[3]))+.14;
}
function silenceAlarm(key) {
  const gains=audioByAlarm.get(key);
  if (gains&&audioContext) for(const gain of gains) {
    try {gain.gain.cancelScheduledValues(audioContext.currentTime);gain.gain.setValueAtTime(0.0001,audioContext.currentTime);}catch{}
  }
  audioByAlarm.delete(key);
}
const keyFor = t => `${t.local?'local':'server'}:${t.id}:${t.created_at_ms}`;
function markSeen(key) {
  let seen={};try{seen=JSON.parse(localStorage.getItem(SEEN_KEY)||'{}')}catch{}
  if(seen[key]) return false;
  const now=Date.now();seen[key]=now;
  try{localStorage.setItem(SEEN_KEY,JSON.stringify(Object.fromEntries(Object.entries(seen).filter(([,at])=>now-at<86400000).slice(-300))))}catch{}
  return true;
}
function persistPending() {
  try{localStorage.setItem(PENDING_KEY,JSON.stringify([...active.values()]
    .filter(a=>a.alarm.repeat==='until_ack').map(({key,name,started,alarm})=>({key,name,started,alarm}))));}catch{}
}
const escapeHTML=v=>String(v).replace(/[&<>"']/g,ch=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
function render() {
  const banner=document.getElementById('alarm-banner'), list=document.getElementById('alarm-list');
  if (!banner || !list) return;
  const pending=[...active.values()].filter(a=>a.alarm.visual || a.alarm.repeat==='until_ack');
  banner.hidden=pending.length===0;
  document.getElementById('alarm-count').textContent=pending.length>1?`(${pending.length})`:'';
  document.getElementById('alarm-ack-all').hidden=!pending.some(a=>a.alarm.repeat==='until_ack');
  list.innerHTML=pending.map(a=>{
    const needsAck=a.alarm.repeat==='until_ack';
    return `<div class="alarm-item ${a.alarm.visual?'alarm-visible':''} ${a.alarm.visual?'alarm-pulsing':''}" data-alarm-key="${escapeHTML(a.key)}">
      <span class="alarm-item-name">${escapeHTML(a.name)}</span><span class="alarm-item-state">Abgelaufen</span>
      ${needsAck?`<button type="button" data-ack="${escapeHTML(a.key)}">Bestätigen</button>`:''}
    </div>`;
  }).join('');
  updateTabAlarm();
  onChange(pending);
}
export function acknowledgeAlarm(key) {
  if(!active.has(key))return;
  silenceAlarm(key);active.delete(key);persistPending();render();
}
export function acknowledgeAll() {
  for (const [key,a] of active)if(a.alarm.repeat==='until_ack'){silenceAlarm(key);active.delete(key);}
  persistPending();render();
}
function soundTick() {
  const now=Date.now();let changed=false;
  for(const [key,a] of active) {
    if(now>=a.started+alarmWindowMs(a.alarm)){silenceAlarm(key);active.delete(key);changed=true;continue;}
    if(a.nextSound!==null&&now>=a.nextSound){
      playFinishSound(a.alarm.tone,a.alarm.volume,a.key);
      a.played++;
      a.nextSound=a.alarm.repeat==='once'||(a.alarm.repeat==='three'&&a.played>=3)?null:now+(a.alarm.repeat==='three'?THREE_INTERVAL_MS:UNTIL_ACK_INTERVAL_MS);
    }
  }
  if(changed){persistPending();render();}
}
async function showSystemNotification(timer,alarm) {
  if (!alarm.system || permission()!=='granted')return;
  const title=`${timer.name} – Fertig!`;
  // The browser's notification chime is not the selected NextTimer tone.
  // Suppress it so users hear only the configured in-app alarm sound.
  const options={body:'Der Countdown ist abgelaufen.',tag:`nexttimer-${keyFor(timer)}`,icon:new URL('../icons/icon-192.png',import.meta.url).href,silent:true};
  try {
    if('serviceWorker' in navigator){
      const reg=await navigator.serviceWorker.getRegistration();
      if(reg?.active){await reg.showNotification(title,options);return;}
    }
    new Notification(title,options);
  }catch(error){console.warn('Timer-Benachrichtigung:',error);}
}
export function alertFinished(timer) {
  const key=keyFor(timer);
  if(!markSeen(key))return;
  const alarm=resolveAlarm(timer),now=Date.now();
  if(alarm.visual||alarm.repeat==='until_ack'||(alarm.tone!=='none' && alarm.volume>0)){
    active.set(key,{key,name:timer.name||'Timer',alarm,started:now,played:0,
      nextSound:alarm.tone==='none'||alarm.volume===0?null:now});
    persistPending();render();soundTick();
  }
  void showSystemNotification(timer,alarm);
}
export function initBrowserAlarms(callback=()=>{}) {
  onChange=callback;
  if (!tabInitialized) {
    normalTitle = document.title;
    iconElement = document.querySelector('link[rel~="icon"]');
    normalIcon = iconElement?.getAttribute('href') ?? '';
    tabInitialized = true;
  }
  try {
    const restored=JSON.parse(localStorage.getItem(PENDING_KEY)||'[]');
    for(const a of Array.isArray(restored)?restored:[]){
      if(typeof a.key==='string'&&typeof a.name==='string'&&valid(a.alarm)&&a.alarm.repeat==='until_ack'&&Date.now()-a.started<86400000){
        active.set(a.key,{...a,alarm:{...a.alarm,volume:Number.isFinite(a.alarm.volume)?clamp(a.alarm.volume):BASE.volume},played:0,nextSound:Date.now()+1000});
      }
    }
  }catch{}
  const banner=document.getElementById('alarm-banner');
  banner?.addEventListener('click',e=>{
    const key=e.target.closest('[data-ack]')?.dataset.ack;
    if(key)acknowledgeAlarm(key);
    if(e.target.closest('#alarm-ack-all'))acknowledgeAll();
  });
  render();
  window.setInterval(soundTick,250);
  window.setInterval(toggleTabTitle,1200);
  // Restore the normal title/icon immediately when the tab becomes visible,
  // and discard expired alarms after background throttling or suspension.
  document.addEventListener('visibilitychange',()=>{soundTick();render();});
}
