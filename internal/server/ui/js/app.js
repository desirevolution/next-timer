import {enableFavoriteSorting} from './favorite-sort.js?v=24';
import {createStandaloneAPI} from './standalone-store.js?v=24';
import {LocalTimers} from './local-timers.js?v=24';
import {alertFinished, permission, requestNotifications, unlockAudio, playFinishSound} from './alerts.js?v=24';

const $ = id => document.getElementById(id);
const STANDALONE = document.documentElement.dataset.mode === 'standalone';
let standaloneStore;
const esc = value => String(value ?? '').replace(/[&<>"']/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
const fmt = ms => {
  const seconds = Math.max(0,Math.ceil(ms/1000));
  const h = Math.floor(seconds/3600), m = Math.floor(seconds%3600/60), s = seconds%60;
  return (h?`${String(h).padStart(2,'0')}:`:'')+`${String(m).padStart(2,'0')}:${String(s).padStart(2,'0')}`;
};
const durationLabel = raw => {
  const value = String(raw), units = [...value.matchAll(/(\d+(?:\.\d+)?)(h|ms|m|s)/g)];
  if (!units.length || units.map(x=>x[0]).join('') !== value) return value;
  return fmt(units.reduce((sum,x)=>sum+Number(x[1])*({h:3600000,m:60000,s:1000,ms:1}[x[2]]),0));
};
const cachedFavorites = (() => {try {return JSON.parse(localStorage.getItem('nexttimer.templates-cache.v1') || '[]');} catch {return [];}})();
let serverState = {timers:[],templates:cachedFavorites,server_now_ms:Date.now()};
let current = {...serverState}, skew = 0, sorting = false, editID = null, renameGroup = null, localTimers;
let bannerTimer;
let renderedFavKey = '', renderedTimerKey = '', renderedFocusID = '', savingOrder = false;
const remaining = t => t.status==='running'?Math.max(0,t.deadline_ms-(Date.now()+(t.local?0:skew))):t.remaining_ms;
const fraction = t => t.duration_ms>0?Math.max(0,Math.min(1,1-remaining(t)/t.duration_ms)):0;
function message(text='',error=false) {const el=$('message');el.textContent=text;el.classList.toggle('error',error);}
function sortStatus(text='',error=false) {const el=$('sort-status');el.textContent=text;el.classList.toggle('error',error);}
async function api(path,method='GET',body) {
  if(STANDALONE) return standaloneStore.api(path,method,body);
  const response = await fetch('/api'+path,{method,headers:body?{'Content-Type':'application/json'}:{},body:body?JSON.stringify(body):undefined});
  if (!response.ok) {const value = await response.json().catch(()=>({error:`HTTP ${response.status}`}));throw Error(value.error||`HTTP ${response.status}`);}
  return response.status===204?null:response.json();
}
async function perform(fn) {
  try {await fn();message();await reload().catch(()=>draw());return true;}
  catch(e){message(e.message,true);return false;}
}
function combine() {
  current={...serverState,timers:[...(serverState.timers||[]),...(localTimers?.all()||[])]};
  // The server's focus may differ: local timers are never sent to AWTRIX.
  draw();
}
function apply(state) {
  serverState=state;
  skew=state.server_now_ms-Date.now();
  try {localStorage.setItem('nexttimer.templates-cache.v1',JSON.stringify(state.templates||[]));} catch {}
  combine();
}
async function reload(){apply(await api('/state'));}
function button(action,label,id,title,extra='') {
  return `<button type="button" data-action="${action}" data-id="${esc(id)}" class="${extra}" aria-label="${esc(title)}" title="${esc(title)}">${label}</button>`;
}
function focusTimer(){
  return (current.timers||[]).filter(t=>t.status==='running')
    .sort((a,b)=>remaining(a)-remaining(b)||a.created_at_ms-b.created_at_ms||a.id.localeCompare(b.id))[0];
}
function drawFocus(){
  const focus=focusTimer();
  $('head-status').textContent=focus?(focus.local&&!STANDALONE?'Lokal · läuft':'Läuft'):'Bereit';
  $('hero-name').textContent=focus?focus.name:'Noch kein Timer aktiv';
  $('hero-origin').hidden=STANDALONE||!focus?.local;
  $('hero-controls').hidden=!focus;
  $('focus-panel').classList.toggle('idle',!focus);
  if (focus?.id !== renderedFocusID) renderedFocusID=focus?.id||'';
  for (const action of ['hero-minus','hero-toggle','hero-plus','hero-stop']) $(action).dataset.id=focus?.id||'';
  if(focus){$('hero-toggle').dataset.action=focus.status==='paused'?'resume':'pause';$('toggle-symbol').textContent=focus.status==='paused'?'▶':'Ⅱ';$('toggle-label').textContent=focus.status==='paused'?'Weiter':'Pause';}
  refreshTime();
}
function refreshTime(){
  const focus=focusTimer();
  const urgent=Boolean(focus && focus.status==='running' && remaining(focus)<=10000);
  $('focus-panel').classList.toggle('urgent',urgent);
  $('hero-time').textContent=focus?fmt(remaining(focus)):'--:--';
  const percentage=focus?Math.round(fraction(focus)*100):0;
  $('hero-bar').style.width=percentage+'%';$('hero-progress').setAttribute('aria-valuenow',String(percentage));
  for(const el of document.querySelectorAll('[data-time]')){
    const t=(current.timers||[]).find(timer=>timer.id===el.dataset.time);
    if(t){el.textContent=fmt(remaining(t));el.classList.toggle('urgent',t.status==='running'&&remaining(t)<=10000);}
  }
}
const groupOf = fav => (fav.group||'').trim();
// The server defines creation order; the default group is always displayed last.
function favoriteGroups() {
  const favs=current.templates||[];
  const names=[...(current.groups||[])];
  for(const fav of favs)if(groupOf(fav)&&!names.some(g=>g.toLowerCase()===groupOf(fav).toLowerCase()))names.push(groupOf(fav));
  return [...names.map(name=>({name,items:favs.filter(f=>groupOf(f).toLowerCase()===name.toLowerCase())})).filter(g=>g.items.length),
    {name:'',items:favs.filter(f=>!groupOf(f))}].filter(g=>g.items.length);
}
function favoriteCard(fav){
  if(!sorting)return `<article class="fav-card" data-id="${esc(fav.id)}"><button class="fav-start" type="button" data-action="start-fav" data-id="${esc(fav.id)}" aria-label="${esc(fav.name)}, ${esc(durationLabel(fav.duration))}, starten"><span class="fav-name">${esc(fav.name)}</span><span class="fav-duration">${esc(durationLabel(fav.duration))}</span>${fav.browser_only&&!STANDALONE?'<span class="fav-local">Nur Browser</span>':''}<span class="fav-play">▶ Starten</span></button></article>`;
  return `<article class="fav-card" data-id="${esc(fav.id)}"><div class="sort-main"><button type="button" class="drag-handle" aria-label="${esc(fav.name)} innerhalb der Gruppe ziehen" title="In der Gruppe verschieben">⠿</button><div><div class="sort-name">${esc(fav.name)}</div><div class="sort-duration">${esc(durationLabel(fav.duration))}${fav.browser_only&&!STANDALONE?' · Lokal':''}</div></div></div><div class="sort-actions">${button('fav-up','↑',fav.id,'Innerhalb der Gruppe zurück')}${button('fav-down','↓',fav.id,'Innerhalb der Gruppe vor')}${button('edit-fav','✎',fav.id,'Favorit bearbeiten')}${button('delete-fav','×',fav.id,'Favorit löschen','delete')}</div></article>`;
}
function drawFavorites(force=false){
  const favs=current.templates||[];
  $('fav-count').textContent=favs.length;
  const key=JSON.stringify([favs,current.groups,sorting]);
  if(!force && renderedFavKey===key)return;
  if(dragSort.isDragging()&&!force)return;
  renderedFavKey=key;
  $('favorites').classList.toggle('sorting',sorting);
  $('sort-info').hidden=!sorting;
  $('sort-toggle').textContent=sorting?'✓ Fertig':'✎ Favoriten bearbeiten';
  $('sort-toggle').setAttribute('aria-pressed',String(sorting));
  $('fav-hint').textContent=sorting?'Vorlagen bearbeiten und innerhalb der Gruppen sortieren':'Ein Klick startet den Timer.';
  const groups=favoriteGroups(), headings=groups.some(g=>g.name!=='');
  $('favorites').innerHTML=favs.length?groups.map(g=>`<section class="favorite-group" data-group="${esc(g.name)}" aria-label="${esc(g.name||'Standard')}">
    ${headings?`<div class="favorite-group-heading"><h3>${esc(g.name||'Standard')}</h3>${sorting&&g.name?`<div class="group-actions"><button type="button" data-action="group-up" data-group="${esc(g.name)}" aria-label="Gruppe ${esc(g.name)} nach oben" title="Gruppe nach oben" ${groups.filter(x=>x.name).findIndex(x=>x.name===g.name)===0?'disabled':''}>↑</button><button type="button" data-action="group-down" data-group="${esc(g.name)}" aria-label="Gruppe ${esc(g.name)} nach unten" title="Gruppe nach unten" ${groups.filter(x=>x.name).findIndex(x=>x.name===g.name)===groups.filter(x=>x.name).length-1?'disabled':''}>↓</button><button type="button" class="group-rename-button" data-action="rename-group" data-group="${esc(g.name)}" aria-label="Gruppe ${esc(g.name)} umbenennen">✎ Umbenennen</button></div>`:''}</div>`:''}
    <div class="favorites-grid">${g.items.map(favoriteCard).join('')}</div></section>`).join(''):'<p class="empty">Noch keine Favoriten. Öffne „Neuen Timer erstellen“, um einen anzulegen.</p>';
  if(sorting)for(const group of $('favorites').querySelectorAll('.favorite-group')){
    const cards=[...group.querySelectorAll('.fav-card')];
    cards.forEach((card,i)=>{
      card.querySelector('[data-action="fav-up"]').disabled=i===0;
      card.querySelector('[data-action="fav-down"]').disabled=i===cards.length-1;
    });
  }
  const opts=(current.groups||[]).map(n=>`<option value="${esc(n)}"></option>`).join('');
  $('favorite-group-options').innerHTML=opts;
}
function drawOthers(){
  // Finished timers are announced by the completion banner, not moved to the active list.
  const timers=(current.timers||[]).filter(t=>t.status!=='finished' && t.id!==focusTimer()?.id);
  $('timer-count').textContent=timers.length;
  const key=JSON.stringify(timers);
  if(key===renderedTimerKey)return;
  renderedTimerKey=key;
  $('timers').innerHTML=timers.length?timers.map(t=>`<div class="timer-row"><div><strong>${esc(t.name)}</strong><div class="timer-meta">${t.local&&!STANDALONE?'Nur in diesem Browser · ':''}${t.status==='finished'?'Fertig':t.status==='paused'?'Pausiert':'Läuft'}</div></div><div class="timer-right"><span class="remaining" data-time="${esc(t.id)}">${fmt(remaining(t))}</span>${t.status==='running'?button('pause','Ⅱ',t.id,'Pausieren'):t.status==='paused'?button('resume','▶',t.id,'Fortsetzen'):''}${t.status!=='finished'?button('adjust','+1m',t.id,'Eine Minute hinzufügen'):''}${button('stop','×',t.id,'Timer beenden','danger')}</div></div>`).join(''):'<p class="empty">Keine weiteren Timer.</p>';
}
function draw(){drawFocus();drawFavorites();drawOthers();}
async function commitOrder(ids){
  if(savingOrder)return;
  const original=(current.templates||[]).map(x=>x.id);
  if(ids.join('|')===original.join('|'))return;
  savingOrder=true;
  sortStatus('Speichere Reihenfolge …');
  try{apply(await api('/templates/order','PUT',{ids}));sortStatus('Reihenfolge gespeichert.');}
  catch(e){sortStatus(e.message,true);await reload().catch(()=>{});}
  finally{savingOrder=false;}
}
const dragSort=enableFavoriteSorting($('favorites'),commitOrder,()=>drawFavorites(true));
function moveFavorite(id,delta){
  if(savingOrder)return;
  const ids=(current.templates||[]).map(t=>t.id),from=ids.indexOf(id);
  if(from<0)return;
  const group=groupOf(current.templates[from]);
  const peers=ids.filter(favId=>groupOf(current.templates.find(t=>t.id===favId))===group);
  const peerIndex=peers.indexOf(id),next=peerIndex+delta;
  if(next<0||next>=peers.length)return;
  const otherID=peers[next],to=ids.indexOf(otherID);
  [ids[from],ids[to]]=[ids[to],ids[from]];
  commitOrder(ids);
}
async function moveGroup(name, delta) {
  if(savingOrder)return;
  const groups=[...(current.groups||[])];
  const index=groups.findIndex(g=>g.toLowerCase()===name.toLowerCase());
  const target=index+delta;
  if(index<0||target<0||target>=groups.length)return;
  [groups[index],groups[target]]=[groups[target],groups[index]];
  savingOrder=true;
  sortStatus('Speichere Gruppenreihenfolge …');
  try {apply(await api('/groups/order','PUT',{groups}));sortStatus('Gruppenreihenfolge gespeichert.');}
  catch(e){sortStatus(e.message,true);await reload().catch(()=>{});}
  finally {savingOrder=false;}
}
function showFinished(timer) {
  $('finished-label').textContent=`„${timer.name}“ ist fertig.`;
  $('finished-banner').hidden=false;
  clearTimeout(bannerTimer);
  bannerTimer=window.setTimeout(()=>{$('finished-banner').hidden=true;},20000);
}
async function timerControl(id, action, deltaText) {
  if(id.startsWith('local-')) {
    let delta=0;
    if(action==='adjust') delta=(deltaText==='-1m'?-1:1)*60000;
    localTimers.control(id,action,delta);
    return;
  }
  if(action==='adjust') return api('/timers/'+encodeURIComponent(id)+'/adjust','POST',{delta:deltaText||'+1m'});
  return api('/timers/'+encodeURIComponent(id)+'/'+action,'POST');
}
const formOptions = prefix => ({
  browser_only:STANDALONE||$(prefix+'browser-only').checked,
  notify:$(prefix+'notify').checked,
  sound:$(prefix+'sound').checked
});
function startTimer(duration,name,templateId,options={}){
  const fav=templateId?(current.templates||[]).find(f=>f.id===templateId):null;
  if(templateId&&!fav) throw Error('Favorit nicht gefunden.');
  const config=fav?{browser_only:STANDALONE||!!fav.browser_only,notify:fav.notify!==false,sound:fav.sound!==false}:{...options,browser_only:STANDALONE||!!options.browser_only};
  if(config.browser_only) return localTimers.start(fav?fav.name:name,fav?fav.duration:duration,config);
  return api('/timers','POST',fav?{template_id:templateId}:{duration,name,...config});
}
$('new-form').addEventListener('submit',e=>{e.preventDefault();perform(()=>startTimer($('duration').value,$('name').value,null,formOptions('timer-')));});
$('save-favorite').addEventListener('click',async()=>{
  const name=$('name').value.trim()||'Timer';
  const ok=await perform(()=>api('/templates','POST',{duration:$('duration').value,name,group:$('timer-group').value.trim(),...formOptions('timer-')}));
  if(ok)$('new-section').open=false;
});
function openFavoriteEditor(fav) {
  editID=fav.id;
  $('edit-name').value=fav.name;
  $('edit-duration').value=fav.duration;
  $('edit-group').value=fav.group||'';
  $('edit-browser-only').checked=STANDALONE||!!fav.browser_only;
  $('edit-notify').checked=fav.notify!==false;
  $('edit-sound').checked=fav.sound!==false;
  $('edit-error').textContent='';
  $('favorite-dialog').showModal();
  $('edit-name').focus();
}
$('edit-cancel').addEventListener('click',()=>$('favorite-dialog').close());
$('favorite-dialog').addEventListener('close',()=>{editID=null;$('edit-error').textContent='';});
$('edit-form').addEventListener('submit',async e=>{
  e.preventDefault();
  if(!editID)return;
  $('edit-submit').disabled=true;
  try {
    await api('/templates/'+encodeURIComponent(editID),'PUT',{
      name:$('edit-name').value.trim(),duration:$('edit-duration').value.trim(),
      group:$('edit-group').value.trim(),browser_only:STANDALONE||$('edit-browser-only').checked,
      notify:$('edit-notify').checked,sound:$('edit-sound').checked
    });
    $('favorite-dialog').close();
    await reload();
    message('Favorit gespeichert.');
  } catch(error) {$('edit-error').textContent=error.message;}
  finally {$('edit-submit').disabled=false;}
});
$('group-cancel').addEventListener('click',()=>$('group-dialog').close());
$('group-dialog').addEventListener('close',()=>{renameGroup=null;$('group-error').textContent='';});
$('group-form').addEventListener('submit',async e=>{
  e.preventDefault();if(!renameGroup)return;
  $('group-submit').disabled=true;
  try {
    apply(await api('/groups/'+encodeURIComponent(renameGroup),'PUT',{name:$('group-rename').value.trim()}));
    $('group-dialog').close();message('Gruppe umbenannt.');
  } catch(error) {$('group-error').textContent=error.message;}
  finally {$('group-submit').disabled=false;}
});
$('sort-toggle').addEventListener('click',()=>{
  if(savingOrder)return;
  sorting=!sorting;
  sortStatus();
  drawFavorites(true);
});
document.addEventListener('click',e=>{
  const button=e.target.closest('button[data-action]');
  if(!button)return;
  const {action,id}=button.dataset;
  if(action==='group-up'||action==='group-down'){moveGroup(button.dataset.group,action==='group-up'?-1:1);return;}
  if(action==='rename-group'){renameGroup=button.dataset.group;$('group-rename').value=renameGroup;$('group-dialog').showModal();$('group-rename').focus();return;}
  if(action==='fav-up'||action==='fav-down'){moveFavorite(id,action==='fav-up'?-1:1);return;}
  if(action==='start-fav'){if(!sorting)perform(()=>startTimer('','',id));return;}
  if(action==='edit-fav'){
    const fav=(current.templates||[]).find(t=>t.id===id);if(!fav)return;
    openFavoriteEditor(fav);return;
  }
  if(action==='delete-fav'){
    const fav=(current.templates||[]).find(t=>t.id===id);
    if(fav&&confirm(`Favorit „${fav.name}“ löschen?`))perform(()=>api('/templates/'+encodeURIComponent(id),'DELETE'));
    return;
  }
  if(action==='adjust'){
    const delta=button.dataset.delta||'+1m';
    perform(()=>timerControl(id,'adjust',delta));return;
  }
  perform(()=>timerControl(id,action));
});
const events=STANDALONE?null:new EventSource('/api/events');
events?.addEventListener('state',e=>{
  try{const event=JSON.parse(e.data);apply(event.state);if(event.type==='finished'&&event.timer)alertFinished(event.timer,showFinished);$('connect').textContent='Verbunden';$('live').classList.remove('off');}
  catch(error){console.error(error);}
});
events?.addEventListener('error',()=>{$('connect').textContent='Verbinde erneut …';$('live').classList.add('off');});
$('finished-close').addEventListener('click',()=>{$('finished-banner').hidden=true;clearTimeout(bannerTimer);});
document.addEventListener('pointerdown', unlockAudio, {passive:true});
function updateNotificationButton(){
  const p=permission();
  const action=$('notifications-enable'),status=$('notifications-status');
  action.hidden=p!=='default';
  status.hidden=p==='default';
  status.textContent=p==='granted'?'✓ Benachrichtigungen aktiv':p==='denied'?'Benachrichtigungen blockiert · Freigabe in den Browser-Einstellungen möglich':'Systembenachrichtigungen hier nicht verfügbar';
}
$('notifications-enable').addEventListener('click',async()=>{
  try {await requestNotifications();updateNotificationButton();}
  catch(error){message(error.message,true);}
});
updateNotificationButton();
localTimers=new LocalTimers(combine,t=>alertFinished(t,showFinished));
if(STANDALONE){
  standaloneStore=createStandaloneAPI(()=>reload().catch(console.warn));
  $('connect').textContent='Nur Browser';
  $('live').classList.remove('off');
  $('timer-browser-only').checked=true;
  $('edit-browser-only').checked=true;
}
combine();
// PWA registration and user-confirmed updates are managed in /pwa-update.js.
window.setInterval(refreshTime,250);
if(!STANDALONE)window.setInterval(()=>{if(events.readyState!==EventSource.OPEN)reload().catch(()=>{});},10000);
reload().catch(()=>{
  if(!STANDALONE)message('Server nicht erreichbar. Lokale Timer funktionieren offline; gespeicherte Favoriten können lokal gestartet werden.',false);
});
if(STANDALONE){
  $('standalone-export')?.addEventListener('click',()=>{
    const data=JSON.stringify(standaloneStore.exportBackup(localTimers.all()),null,2);
    const url=URL.createObjectURL(new Blob([data],{type:'application/json'}));
    const a=document.createElement('a');a.href=url;a.download='nexttimer-backup.json';a.click();
    setTimeout(()=>URL.revokeObjectURL(url),1000);
  });
  $('standalone-import')?.addEventListener('change',async e=>{
    const file=e.target.files?.[0];if(!file)return;
    try{
      const data=standaloneStore.importBackup(JSON.parse(await file.text()));
      if(!confirm('Favoriten und lokale Timer durch die Sicherung ersetzen?'))return;
      standaloneStore.replaceFavorites(data.favorites);
      localStorage.setItem('nexttimer.local-timers.v1',JSON.stringify(data.timers));
      localTimers.timers=localTimers.read();
      await reload();message('Sicherung importiert.');
    }catch(err){message(err.message,true);}
    finally{e.target.value='';}
  });
}
