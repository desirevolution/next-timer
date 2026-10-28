// Local/browser notifications are best effort. No guaranteed alarms after app shutdown.
const KEY = 'nexttimer.alerted.v1';
let audioContext;
export function canNotify() { return 'Notification' in window && window.isSecureContext; }
export function permission() { return canNotify() ? Notification.permission : 'unsupported'; }
export async function requestNotifications() {
  if (!canNotify()) throw Error('Systembenachrichtigungen benötigen HTTPS oder localhost und einen unterstützten Browser.');
  return Notification.requestPermission();
}
export function unlockAudio() {
  try {
    audioContext ??= new (window.AudioContext || window.webkitAudioContext)();
    if (audioContext.state === 'suspended') audioContext.resume().catch(()=>{});
  } catch {}
}
// A deliberately noticeable original three-tone double chime (~1.7s).
// No audio assets are fetched; the Web Audio API produces the signal offline.
export function playFinishSound() {
  try {
    unlockAudio();
    if (!audioContext || audioContext.state !== 'running') return;
    const start=audioContext.currentTime+0.015;
    const tones=[{t:0,f:740},{t:.23,f:988},{t:.47,f:1318},{t:.93,f:740},{t:1.16,f:988},{t:1.4,f:1318}];
    for (const {t,f} of tones) {
      const osc=audioContext.createOscillator(),gain=audioContext.createGain();
      osc.type='triangle';
      osc.frequency.setValueAtTime(f,start+t);
      gain.gain.setValueAtTime(0.0001,start+t);
      gain.gain.exponentialRampToValueAtTime(0.23,start+t+.025);
      gain.gain.exponentialRampToValueAtTime(0.0001,start+t+.19);
      osc.connect(gain); gain.connect(audioContext.destination);
      osc.start(start+t); osc.stop(start+t+.20);
    }
  } catch (error) { console.warn('Signalton:',error); }
}
function newlyFinished(timer) {
  const key = `${timer.local?'local':'server'}:${timer.id}:${timer.created_at_ms}`;
  let seen={};
  try {seen=JSON.parse(localStorage.getItem(KEY)||'{}');} catch {}
  if (seen[key]) return false;
  seen[key]=Date.now();
  const entries=Object.entries(seen).filter(([,at])=>Date.now()-at<86400000).slice(-200);
  try {localStorage.setItem(KEY,JSON.stringify(Object.fromEntries(entries)));} catch {}
  return true;
}
export async function alertFinished(timer, showInApp) {
  if (!newlyFinished(timer)) return;
  if (timer.notify!==false) showInApp(timer);
  if (timer.sound!==false) playFinishSound();
  if (timer.notify===false || permission()!=='granted') return;
  const title=`${timer.name} – Fertig!`;
  const opts={body:'Der Countdown ist abgelaufen.',tag:`nexttimer-${timer.local?'local':'server'}-${timer.id}-${timer.created_at_ms}`,icon:'/icons/icon-192.png'};
  try {
    if ('serviceWorker' in navigator) {
      const reg=await navigator.serviceWorker.getRegistration();
      if (reg?.active) await reg.showNotification(title,opts);
      else new Notification(title,opts);
    } else new Notification(title,opts);
  } catch (error) {console.warn('Timer-Benachrichtigung:',error);}
}
