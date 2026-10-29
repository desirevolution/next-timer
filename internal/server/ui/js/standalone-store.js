// Browser-only favorite storage. The public API mirrors the small set of
// template endpoints used by the shared NextTimer frontend; no HTTP is sent.
import {parseTimerDuration} from './local-timers.js?v=26';

const KEY = 'nexttimer.standalone.templates.v1';
const BACKUP_VERSION = 1;
const same = (a,b) => a.toLocaleLowerCase() === b.toLocaleLowerCase();
const copy = value => JSON.parse(JSON.stringify(value));
const id = () => `f-${crypto.randomUUID?.() || Date.now()+'-'+Math.random().toString(36).slice(2)}`;
function normalizeGroup(input) {
  const group = String(input||'').trim();
  if (same(group, 'Standard')) return '';
  if ([...group].length > 48 || /[\x00-\x1f\x7f]/.test(group)) throw Error('Ungültiger Gruppenname (maximal 48 Zeichen).');
  return group;
}
function normalizeAlarm(raw) {
  if (raw == null) return null;
  if (!raw || typeof raw !== 'object' || !['none','gong','bell','digital','alert'].includes(raw.tone) ||
      !['once','three','until_ack'].includes(raw.repeat) || typeof raw.visual !== 'boolean' || typeof raw.system !== 'boolean')
    throw Error('Ungültige Alarmeinstellungen.');
  if (raw.volume !== undefined && (typeof raw.volume !== 'number' || !Number.isFinite(raw.volume) || raw.volume < 0 || raw.volume > 1))
    throw Error('Lautstärke muss zwischen 0 und 100 % liegen.');
  return {tone:raw.tone,repeat:raw.repeat,visual:raw.visual,system:raw.system,
    ...(raw.volume === undefined ? {} : {volume:raw.volume})};
}
function normalizeFavorite(raw, existing) {
  const name = String(raw.name||'').trim();
  if (!name || [...name].length > 80) throw Error('Favoritenname muss 1–80 Zeichen haben.');
  const ms = parseTimerDuration(raw.duration);
  const group = normalizeGroup(raw.group);
  return {id:existing?.id || id(), name, duration:`${ms}ms`, group,
    group_order:existing?.group_order||0, browser_only:true,
    sound:raw.sound!==false,notify:raw.notify!==false,alarm:normalizeAlarm(raw.alarm)};
}
function orderItems(items) {
  items.sort((a,b)=>{
    if (!a.group && !b.group) return 0;
    if (!a.group) return 1;
    if (!b.group) return -1;
    return a.group_order-b.group_order;
  });
  return items;
}
function groupsOf(items) {
  return [...new Set(items.filter(x=>x.group).map(x=>x.group))];
}
function validateItems(items) {
  if (!Array.isArray(items) || items.length>500) throw Error('Ungültige Favoritendaten.');
  const result=[];
  const ids=new Set();
  for(const raw of items) {
    const t=normalizeFavorite(raw,{id:raw.id,group_order:raw.group_order});
    if(typeof t.id!=='string'||!t.id||ids.has(t.id)) throw Error('Doppelte oder ungültige Favoriten-ID.');
    ids.add(t.id);
    if(result.some(f=>same(f.group,t.group)&&same(f.name,t.name))) throw Error('Name in dieser Gruppe bereits vergeben.');
    result.push(t);
  }
  // Preserve validated group positions, normalizing to a contiguous order.
  const names=[...new Set(result.filter(x=>x.group).sort((a,b)=>a.group_order-b.group_order).map(x=>x.group.toLocaleLowerCase()))];
  for(const item of result) if(item.group) item.group_order=names.indexOf(item.group.toLocaleLowerCase())+1;
  return orderItems(result);
}

export function createStandaloneAPI(onExternalChange=()=>{}) {
  let items=load();
  function load() {
    try {return validateItems(JSON.parse(localStorage.getItem(KEY)||'[]'));}
    catch(error) {console.warn('NextTimer standalone favorites:',error);return [];}
  }
  function save(next) {
    // Don't update in-memory state unless storage has succeeded.
    const valid=validateItems(next);
    try {localStorage.setItem(KEY,JSON.stringify(valid));}
    catch {throw Error('Favoriten konnten nicht gespeichert werden. Prüfe den Browserspeicher.');}
    items=valid;
  }
  window.addEventListener('storage',event=>{
    if(event.key===KEY) {items=load();onExternalChange();}
  });
  function snapshot(){return {server_now_ms:Date.now(),timers:[],templates:copy(items),groups:groupsOf(items)};}
  function ensureUnique(fav,excludedID='') {
    if(items.some(t=>t.id!==excludedID&&same(t.group,fav.group)&&same(t.name,fav.name)))
      throw Error('Name in dieser Gruppe bereits vergeben.');
  }
  function nextOrder(){return Math.max(0,...items.map(t=>t.group_order||0))+1;}
  function groupInfo(group){return items.find(t=>group&&same(t.group,group));}
  async function api(path,method='GET',body={}) {
    if(path==='/state'&&method==='GET') return snapshot();
    if(path==='/templates'&&method==='POST') {
      const fav=normalizeFavorite(body);
      ensureUnique(fav);
      const found=groupInfo(fav.group);
      fav.group=found?.group||fav.group;
      fav.group_order=found?.group_order|| (fav.group?nextOrder():0);
      save([...items,fav]);return copy(fav);
    }
    if(path==='/templates/order'&&method==='PUT') {
      const ids=body.ids;
      if(!Array.isArray(ids)||ids.length!==items.length||new Set(ids).size!==items.length||ids.some(x=>!items.some(t=>t.id===x))) throw Error('Favoritenliste wurde verändert.');
      const ordered=ids.map(x=>items.find(t=>t.id===x));
      const expected=orderItems([...ordered]);
      if(ordered.some((x,i)=>x.id!==expected[i].id))throw Error('Favoriten lassen sich nur innerhalb einer Gruppe verschieben.');
      save(ordered);return snapshot();
    }
    if(path==='/groups/order'&&method==='PUT') {
      const names=body.groups;
      const current=groupsOf(items);
      if(!Array.isArray(names)||names.length!==current.length||new Set(names.map(x=>String(x).toLowerCase())).size!==names.length||names.some(n=>!current.some(c=>same(c,n)))) throw Error('Gruppenliste wurde verändert.');
      save(orderItems(items.map(t=>({...t,group_order:t.group?names.findIndex(n=>same(n,t.group))+1:0}))));return snapshot();
    }
    if(path.startsWith('/groups/')&&method==='PUT') {
      const old=decodeURIComponent(path.slice('/groups/'.length));
      const next=normalizeGroup(body.name);
      if(!old||!next||!groupInfo(old))throw Error('Gruppe nicht gefunden.');
      if(!same(old,next)&&groupInfo(next))throw Error('Gruppenname bereits vergeben.');
      save(items.map(t=>same(t.group,old)?{...t,group:next}:t));return snapshot();
    }
    const match=path.match(/^\/templates\/([^/]+)$/);
    if(match) {
      const target=decodeURIComponent(match[1]);
      const old=items.find(t=>t.id===target);
      if(!old)throw Error('Favorit nicht gefunden.');
      if(method==='DELETE'){save(items.filter(t=>t.id!==target));return null;}
      if(method==='PUT') {
        const fav=normalizeFavorite(body,old);
        ensureUnique(fav,target);
        const oldGroup=old.group;
        if(!same(fav.group,oldGroup)){
          const existing=groupInfo(fav.group);
          fav.group=existing?.group||fav.group;
          fav.group_order=existing?.group_order||(fav.group?nextOrder():0);
          save([...items.filter(t=>t.id!==target),fav]);
        } else {
          fav.group=oldGroup;
          fav.group_order=old.group_order;
          save(items.map(t=>t.id===target?fav:t));
        }
        return copy(fav);
      }
    }
    throw Error('Dieser Befehl ist im Browser-Modus nicht verfügbar.');
  }
  function exportBackup(localTimers=[]) {
    return {format:'nexttimer-standalone',version:BACKUP_VERSION,exported_at:new Date().toISOString(),favorites:copy(items),timers:copy(localTimers)};
  }
  function importBackup(raw) {
    if(!raw ||raw.format!=='nexttimer-standalone'||raw.version!==BACKUP_VERSION) throw Error('Keine gültige NextTimer-Standalone-Sicherung.');
    const imported=validateItems(raw.favorites);
    if(!Array.isArray(raw.timers)||raw.timers.length>100||raw.timers.some(t=>!t||typeof t.id!=='string'||!t.id.startsWith('local-')||!['running','paused','finished'].includes(t.status)||!Number.isFinite(t.duration_ms)||!Number.isFinite(t.remaining_ms)||!Number.isFinite(t.deadline_ms)))throw Error('Ungültige Timerdaten.');
    return {favorites:imported,timers:raw.timers};
  }
  return {api, snapshot, exportBackup, importBackup, replaceFavorites:save};
}
