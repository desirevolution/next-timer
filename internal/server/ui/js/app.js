import {RecentTimers} from './recent-timers.js?v=40';
import {enableFavoriteSorting} from './favorite-sort.js?v=40';
import {createStandaloneAPI} from './standalone-store.js?v=40';
import {LocalTimers} from './local-timers.js?v=40';
import {parseQuickTimer} from './quick-timer.js?v=40';
import {alertFinished, initBrowserAlarms, getDefaults, saveDefaults, permission, requestNotifications, unlockAudio, playFinishSound} from './alerts.js?v=40';

const $ = id => document.getElementById(id);
const STANDALONE = document.documentElement.dataset.mode === 'standalone';
const L = (text, args={}) => window.NextTimerI18n?.t(text, args) ?? text.replace(/\{(\w+)\}/g,(_,key)=>String(args[key]??''));
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
let previousOtherCount = 0;
const recentTimers = new RecentTimers();
let recentOpen = false, recentIndex = -1, recentDraft = '', recentChoice = null;
// null means automatic selection (soonest running timer); otherwise pin the
// explicitly selected timer until it disappears or finishes.
let manualFocusID = null;

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
  if(typeof drawRecent==='function')drawRecent();
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
function navigableTimers() {
  // Stable carousel order: pausing or adjusting a countdown must never
  // silently reshuffle the position indicator or the navigation buttons.
  return (current.timers||[]).filter(t=>t.status==='running'||t.status==='paused')
    .sort((a,b)=>a.created_at_ms-b.created_at_ms || a.id.localeCompare(b.id));
}
function focusTimer() {
  const timers=navigableTimers();
  if (manualFocusID && !timers.some(t=>t.id===manualFocusID)) manualFocusID=null;
  if(manualFocusID) return timers.find(t=>t.id===manualFocusID);
  // Automatic mode is intentionally independent of carousel/creation order.
  return timers.filter(t=>t.status==='running')
    .sort((a,b)=>remaining(a)-remaining(b)||a.created_at_ms-b.created_at_ms||a.id.localeCompare(b.id))[0]
    || timers[0]; // If everything is paused, show the first paused timer.
}
function selectTimer(offset) {
  const timers=navigableTimers();
  if(!timers.length) return;
  const index=Math.max(0,timers.findIndex(t=>t.id===focusTimer()?.id));
  manualFocusID=timers[(index+offset+timers.length)%timers.length].id;
  draw();
}
function drawFocus(){
  const focus=focusTimer();
  const timers=navigableTimers();
  // The timer display is informational, never a focusable control.
  // Its navigation and action buttons remain individually focusable.
  $('head-status').textContent=focus?(focus.status==='paused'?'Pausiert':focus.local&&!STANDALONE?'Lokal · läuft':'Läuft'):'Bereit';
  const index=focus?timers.findIndex(t=>t.id===focus.id):-1;
  $('focus-position').textContent=focus?`${index+1} von ${timers.length}`:'';
  $('focus-previous').hidden=$('focus-next').hidden=timers.length<2;
  $('hero-name').toggleAttribute('data-i18n-ignore',!!focus);
  $('hero-name').textContent=focus?focus.name:'Noch kein Timer aktiv';
  $('hero-origin').hidden=STANDALONE||!focus?.local;
  $('hero-controls').hidden=!focus;
  $('focus-panel').classList.toggle('idle',!focus);
  if (focus?.id !== renderedFocusID) renderedFocusID=focus?.id||'';
  for (const action of ['hero-minus','hero-toggle','hero-plus','hero-stop','hero-favorite']) $(action).dataset.id=focus?.id||'';
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
  const badge=fav.shortcut?`<span class="favorite-shortcut" title="${esc(L('Mit #{number} in der Eingabe starten',{number:fav.shortcut}))}">#${esc(fav.shortcut)}</span>`:'';
  if(!sorting)return `<article class="fav-card" data-id="${esc(fav.id)}"><button class="fav-start" type="button" data-action="start-fav" data-id="${esc(fav.id)}" aria-label="${esc(fav.name)}, ${esc(durationLabel(fav.duration))}${fav.shortcut?', Schnellstart #'+esc(fav.shortcut):''}, starten"><span class="fav-name" data-i18n-ignore>${esc(fav.name)}</span>${badge}<span class="fav-duration">${esc(durationLabel(fav.duration))}</span>${fav.browser_only&&!STANDALONE?'<span class="fav-local">Nur Browser</span>':''}</button></article>`;
  return `<article class="fav-card" data-id="${esc(fav.id)}"><div class="sort-main"><button type="button" class="drag-handle" aria-label="${esc(L('Favorit {name} zum Sortieren ziehen',{name:fav.name}))}" title="${esc(L('Am Griff ziehen'))}">⠿</button><div class="sort-details"><div class="sort-name"><span data-i18n-ignore>${esc(fav.name)}</span> ${badge}</div><div class="sort-duration">${esc(durationLabel(fav.duration))}${fav.browser_only&&!STANDALONE?' · Lokal':''}</div></div></div><div class="sort-actions">${button('edit-fav','✎',fav.id,'Favorit bearbeiten')}${button('delete-fav','×',fav.id,'Favorit löschen','delete')}</div></article>`;
}
function drawFavorites(force=false){
  const favs=current.templates||[];
  $('fav-count').textContent=favs.length;
  const key=JSON.stringify([favs,current.groups,sorting]);
  if(!force && renderedFavKey===key)return;
  if(dragSort.isDragging()&&!force)return;
  renderedFavKey=key;
  $('favorites').classList.toggle('sorting',sorting);
  $('sort-toggle').textContent=sorting?'✓ Fertig':'✎ Favoriten bearbeiten';
  $('sort-toggle').setAttribute('aria-pressed',String(sorting));
  $('fav-hint').hidden=!sorting;
  $('fav-hint').textContent=sorting?'Am Griff ziehen, um die Reihenfolge zu ändern.':'';
  const groups=favoriteGroups(), headings=groups.some(g=>g.name!=='');
  $('favorites').innerHTML=favs.length?groups.map(g=>`<section class="favorite-group" data-group="${esc(g.name)}" aria-label="${esc(g.name||'Standard')}">
    ${headings?`<div class="favorite-group-heading"><div class="group-label"><h3 ${g.name?'data-i18n-ignore':''}>${esc(g.name||'Standard')}</h3><span class="group-count">${g.items.length}</span></div><span class="group-rule" aria-hidden="true"></span>${sorting&&g.name?`<div class="group-actions"><button type="button" data-action="new-fav-group" data-group="${esc(g.name)}" aria-label="Favorit in Gruppe ${esc(g.name)} hinzufügen" title="Favorit hinzufügen">＋</button><button type="button" data-action="group-up" data-group="${esc(g.name)}" aria-label="Gruppe ${esc(g.name)} nach oben" title="Gruppe nach oben" ${groups.filter(x=>x.name).findIndex(x=>x.name===g.name)===0?'disabled':''}>↑</button><button type="button" data-action="group-down" data-group="${esc(g.name)}" aria-label="Gruppe ${esc(g.name)} nach unten" title="Gruppe nach unten" ${groups.filter(x=>x.name).findIndex(x=>x.name===g.name)===groups.filter(x=>x.name).length-1?'disabled':''}>↓</button><button type="button" class="group-rename-button" data-action="rename-group" data-group="${esc(g.name)}" aria-label="Gruppe ${esc(g.name)} umbenennen">✎ Umbenennen</button></div>`:''}</div>`:''}
    <div class="favorites-grid">${g.items.map(favoriteCard).join('')}</div></section>`).join(''):'<p class="empty">Noch keine Favoriten. Lege deinen ersten Favoriten an.</p>';
  const opts=(current.groups||[]).map(n=>`<option value="${esc(n)}"></option>`).join('');
  $('favorite-group-options').innerHTML=opts;
  renderGroupPicks();
}
function renderGroupPicks() {
  const groups=favoriteGroups().filter(g=>g.name).map(g=>g.name);
  for (const prefix of ['edit']) {
    const field=$(prefix+'-group'), container=$(prefix+'-group-quick');
    container.hidden=groups.length===0;
    container.innerHTML=groups.map(name=>`<button type="button" class="group-quick-button" data-group-choice="${esc(prefix)}" data-group-name="${esc(name)}" aria-pressed="${String(field.value.trim().toLocaleLowerCase()===name.toLocaleLowerCase())}" data-i18n-ignore>${esc(name)}</button>`).join('');
  }
}
function highlightGroupPicks(prefix) {
  const selected=$(prefix+'-group').value.trim().toLocaleLowerCase();
  for(const button of $(prefix+'-group-quick').querySelectorAll('[data-group-choice]')) {
    button.setAttribute('aria-pressed',String(button.dataset.groupName.toLocaleLowerCase()===selected));
  }
}
$('edit-group').addEventListener('input',()=>highlightGroupPicks('edit'));
function drawOthers(){
  // Finished timers are announced by the completion banner, not moved to the active list.
  const timers=navigableTimers().filter(t=>t.id!==focusTimer()?.id);
  $('timer-count').textContent=timers.length;
  if (timers.length === 0) $('others-section').open = false;
  else if (previousOtherCount === 0) $('others-section').open = true;
  previousOtherCount = timers.length;
  const key=JSON.stringify(timers);
  if(key===renderedTimerKey)return;
  renderedTimerKey=key;
  $('timers').innerHTML=timers.length?timers.map(t=>`<div class="timer-row"><button type="button" class="timer-show" data-action="show-timer" data-id="${esc(t.id)}" title="Diesen Timer groß anzeigen"><strong data-i18n-ignore>${esc(t.name)}</strong><span class="timer-meta">${t.local&&!STANDALONE?'Nur in diesem Browser · ':''}${t.status==='paused'?'Pausiert':'Läuft'} · Anzeigen</span></button><div class="timer-right"><span class="remaining" data-time="${esc(t.id)}">${fmt(remaining(t))}</span>${t.status==='running'?button('pause','Ⅱ',t.id,'Pausieren'):t.status==='paused'?button('resume','▶',t.id,'Fortsetzen'):''}${t.status!=='finished'?button('adjust','+1m',t.id,'Eine Minute hinzufügen'):''}${button('save-as-favorite','☆',t.id,'Als Favorit speichern','timer-favorite-icon')}${button('stop','×',t.id,'Timer beenden','danger')}</div></div>`).join(''):'<p class="empty">Keine weiteren Timer.</p>';
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
function profileForm(prefix,alarm=null) {
  const defaults=getDefaults();
  const custom=alarm!==null && alarm!==undefined;
  $(prefix+'alarm-custom').checked=custom;
  const value={...defaults,...alarm};
  $(prefix+'tone').value=value.tone;
  $(prefix+'repeat').value=value.repeat;
  $(prefix+'visual').checked=value.visual;
  $(prefix+'system').checked=value.system;
  $(prefix+'volume').value=Math.round(value.volume*100);
  $(prefix+'volume-label').value=`${Math.round(value.volume*100)} %`;
  $(prefix+'alarm-fields').hidden=!custom;
}
function profileValues(prefix) {
  if(!$(prefix+'alarm-custom').checked)return null;
  return {tone:$(prefix+'tone').value,repeat:$(prefix+'repeat').value,
    visual:$(prefix+'visual').checked,system:$(prefix+'system').checked,
    volume:Number($(prefix+'volume').value)/100};
}
function formOptions(prefix) {
  return {browser_only:STANDALONE||$(prefix+'browser-only').checked,alarm:profileValues(prefix)};
}
const globalControls=['tone','repeat','visual','system','volume'];
function showGlobalDefaults(){
  const v=getDefaults();
  for(const key of globalControls){const el=$('global-'+key);if(key==='visual'||key==='system')el.checked=v[key];else el.value=key==='volume'?Math.round(v.volume*100):v[key];}
  $('global-volume-label').value=`${Math.round(v.volume*100)} %`;
}
function saveGlobalControls(){
  const v={tone:$('global-tone').value,repeat:$('global-repeat').value,
    visual:$('global-visual').checked,system:$('global-system').checked,volume:Number($('global-volume').value)/100};
  try{saveDefaults(v);$('global-volume-label').value=`${Math.round(v.volume*100)} %`;}
  catch(e){message(e.message,true);}
}
for(const key of globalControls) $('global-'+key).addEventListener('input',saveGlobalControls);
for(const prefix of ['timer-','edit-']){
  $(prefix+'alarm-custom').addEventListener('change',()=>{
    if($(prefix+'alarm-custom').checked)profileForm(prefix,{
      tone:$(prefix+'tone').value,repeat:$(prefix+'repeat').value,
      visual:$(prefix+'visual').checked,system:$(prefix+'system').checked,
      volume:Number($(prefix+'volume').value)/100
    });
    else profileForm(prefix);
  });
}
document.querySelectorAll('[data-preview]').forEach(btn=>btn.addEventListener('click',()=>{
  const prefix=btn.dataset.preview+'-';
  playFinishSound($(prefix+'tone').value,Number($(prefix+'volume').value)/100);
}));
for (const prefix of ['timer-','edit-','global-']) $(prefix+'volume').addEventListener('input',()=>{
  $(prefix+'volume-label').value=`${$(prefix+'volume').value} %`;
});
showGlobalDefaults();profileForm('timer-');profileForm('edit-');
async function startTimer(duration,name,templateId,options={}){
  const fav=templateId?(current.templates||[]).find(f=>f.id===templateId):null;
  if(templateId&&!fav) throw Error('Favorit nicht gefunden.');
  const config=fav?{browser_only:STANDALONE||!!fav.browser_only,notify:fav.notify!==false,sound:fav.sound!==false,alarm:fav.alarm??null}:{...options,browser_only:STANDALONE||!!options.browser_only};
  const started=config.browser_only
    ? localTimers.start(fav?fav.name:name,fav?fav.duration:duration,config)
    : await api('/timers','POST',fav?{template_id:templateId}:{duration,name,...config});
  // A successful start is recorded with its own configuration, independent of
  // future edits or deletion of the originating favorite.
  // The server saves global starts for all connected clients; only browser-only
  // starts need to be recorded here in localStorage.
  if(config.browser_only)recentTimers.add({duration:fav?fav.duration:duration,name:fav?fav.name:name,
    options:config});
  drawRecent();
  // An intentional new start must take precedence over an older pinned timer,
  // including a paused one. Do not override manual navigation afterwards.
  if(started?.id) {
    if(!config.browser_only) await reload().catch(()=>{});
    if((current.timers||[]).some(timer=>timer.id===started.id)) {
      manualFocusID=started.id;
      draw();
    }
  }
  return started;
}
// Quick start is a separate, one-line path. The advanced form remains available
// for per-timer alarm settings and saving a favorite.
const quickInput = $('quick-input');
const quickButton = $('quick-start');
const quickError = $('quick-error');
// Only focus a physical-keyboard desktop automatically. On touch devices,
// focusing the input would unnecessarily open the on-screen keyboard.
const desktopKeyboard = () => window.matchMedia?.('(hover: hover) and (pointer: fine)').matches ?? false;
function focusQuickInput(force=false) {
  if ((force || desktopKeyboard()) && !document.querySelector('dialog[open]')) quickInput.focus({preventScroll:true});
}
function setQuickValue(value) {
  quickInput.value=value;
  quickInput.dispatchEvent(new Event('input',{bubbles:true}));
}
function recentEntries() {
  const local=recentTimers.all();
  if(STANDALONE)return local;
  const server=(current.recent||[]).map(v=>({duration:v.duration,name:v.name,
    at:v.started_at_ms-skew,options:{browser_only:false,sound:v.sound,notify:v.notify,alarm:v.alarm||null}}));
  return [...local,...server].sort((a,b)=>(b.at||0)-(a.at||0))
    .filter((v,i,all)=>all.findIndex(x=>JSON.stringify([x.duration,x.name,x.options])===JSON.stringify([v.duration,v.name,v.options]))===i).slice(0,5);
}
function drawRecent() {
  const entries=recentEntries();
  const list=$('recent-list');
  $('recent-toggle').setAttribute('aria-expanded',String(recentOpen));
  $('recent-toggle').disabled=!entries.length;
  $('recent-popover').hidden=!recentOpen||!entries.length;
  list.innerHTML=entries.map((entry,i)=>`<button type="button" class="recent-entry${recentIndex===i?' selected':''}" data-recent-index="${i}" aria-label="${esc(L('Timer {name} erneut starten',{name:entry.name||durationLabel(entry.duration)}))}"><span class="recent-entry-name" data-i18n-ignore>${esc(entry.name||L('Ohne Namen'))}</span><span class="recent-entry-duration">${esc(durationLabel(entry.duration))}</span><span class="recent-entry-play" aria-hidden="true">▶</span></button>`).join('');
}
function closeRecent() {recentOpen=false;drawRecent();}
function resetRecentSelection(){recentIndex=-1;recentChoice=null;}
function stepRecent(direction){
  const entries=recentEntries();
  if(!entries.length)return false;
  if(recentIndex===-1){
    if(quickInput.value.trim())return false;
    recentDraft=quickInput.value;
    if(direction>0)return false;
    recentIndex=0;
  }else{
    recentIndex+= direction < 0 ? 1 : -1;
    if(recentIndex>=entries.length)recentIndex=entries.length-1;
    if(recentIndex<0){resetRecentSelection();setQuickValue(recentDraft);drawRecent();return true;}
  }
  recentChoice=entries[recentIndex];
  setQuickValue(`${recentChoice.duration}${recentChoice.name?' '+recentChoice.name:''}`);
  drawRecent();
  return true;
}
$('recent-toggle').addEventListener('click',()=>{recentOpen=!recentOpen;resetRecentSelection();drawRecent();});
$('recent-list').addEventListener('click',async event=>{
  const target=event.target.closest('[data-recent-index]');if(!target)return;
  const entry=recentEntries()[Number(target.dataset.recentIndex)];if(!entry)return;
  closeRecent();resetRecentSelection();
  if(await perform(()=>startTimer(entry.duration,entry.name,null,entry.options))){
    if(document.activeElement instanceof HTMLElement)document.activeElement.blur();
  }
});
$('recent-clear').addEventListener('click',async()=>{
  recentTimers.clear();
  if(!STANDALONE){try{await api('/recent','DELETE');await reload();}catch(e){message(e.message,true);}}
  closeRecent();resetRecentSelection();
});
quickInput.addEventListener('keydown',event=>{
  if(event.isComposing||event.altKey||event.ctrlKey||event.metaKey||event.shiftKey)return;
  if((event.key==='ArrowUp'||event.key==='ArrowDown')&&stepRecent(event.key==='ArrowUp'?-1:1))event.preventDefault();
});
document.addEventListener('pointerdown',event=>{
  if(recentOpen&&!event.target.closest('.quick-history'))closeRecent();
},{passive:true});
drawRecent();
let quickPending = false;
function clearQuickError() {
  quickError.textContent = '';
  quickInput.removeAttribute('aria-invalid');
}
quickInput.addEventListener('input', () => {
  clearQuickError();
  if(recentChoice && quickInput.value!==`${recentChoice.duration}${recentChoice.name?' '+recentChoice.name:''}`)resetRecentSelection();
  if (!quickPending) quickButton.disabled = !quickInput.value.trim();
});
$('quick-form').addEventListener('submit', async event => {
  event.preventDefault();
  if (quickPending) return;
  clearQuickError();
  let entry, favoriteID=null;
  try {
    const value=quickInput.value.trim();
    if(/^#[0-9]$/.test(value)) {
      const favorite=(current.templates||[]).find(f=>f.shortcut===value.slice(1));
      if(!favorite) throw Error(`Kein Favorit für ${value} zugewiesen.`);
      favoriteID=favorite.id;
    } else if (value.startsWith('#')) {
      throw Error('Favorit mit #1 bis #0 aufrufen.');
    } else entry=parseQuickTimer(value);
  }
  catch (error) {
    quickError.textContent = error.message;
    quickInput.setAttribute('aria-invalid', 'true');
    quickInput.focus();
    return;
  }
  quickPending = true;
  quickButton.disabled = true;
  try {
    if(favoriteID) await startTimer('','',favoriteID);
    else await startTimer(entry.duration, entry.name, null, recentChoice?recentChoice.options:formOptions('timer-'));
    resetRecentSelection();
    closeRecent();
    quickInput.value = '';
    $('quick-options').open = false;
    $('timer-browser-only').checked = STANDALONE;
    profileForm('timer-');
    message();
    // Once a timer is started the global timer shortcuts should work right
    // away. Do not steal focus from controls the user navigated to meanwhile.
    if (document.activeElement === quickInput) quickInput.blur();
    await reload().catch(() => draw());
  } catch (error) {
    quickError.textContent = error.message;
    quickInput.setAttribute('aria-invalid', 'true');
  } finally {
    quickPending = false;
    quickButton.disabled = !quickInput.value.trim();
  }
});
document.querySelectorAll('[data-quick-duration]').forEach(button => button.addEventListener('click', () => {
  resetRecentSelection();
  quickInput.value = button.dataset.quickDuration;
  quickInput.dispatchEvent(new Event('input', {bubbles:true}));
  quickInput.focus();
  quickInput.setSelectionRange(quickInput.value.length,quickInput.value.length);
}));
$('focus-previous').addEventListener('click',()=>selectTimer(-1));
$('focus-next').addEventListener('click',()=>selectTimer(1));
function durationForEditor(ms){
  if(!Number.isFinite(ms))return '';
  if(ms % 3600000===0)return `${ms/3600000}h`;
  if(ms % 60000===0)return `${ms/60000}m`;
  if(ms % 1000===0)return `${ms/1000}s`;
  return `${ms}ms`;
}
$('new-favorite').addEventListener('click',()=>openFavoriteEditor());
$('settings-repair-link').href=STANDALONE?'pwa-repair.html':'/api/pwa-repair';
function openFavoriteEditor(fav=null,{group="",timer=null}={}) {
  editID=fav?.id||null;
  const source=fav||timer;
  $('dialog-title').textContent=fav?'Favorit bearbeiten':'Neuer Favorit';
  $('edit-submit').textContent=fav?'Änderungen speichern':'Favorit speichern';
  $('edit-name').value=source?.name==='Timer'&&!fav?'':source?.name||'';
  $('edit-duration').value=source ? (source.duration || durationForEditor(source.duration_ms)) : '';
  $('edit-group').value=fav?.group||group;
  $('edit-shortcut').value=fav?.shortcut||'';
  highlightGroupPicks('edit');
  $('edit-browser-only').checked=STANDALONE||!!source?.browser_only||!!source?.local;
  const defaults=getDefaults();
  const legacy=(!source?.alarm&&(source?.sound===false||source?.notify===false))?{
    tone:source.sound===false?'none':defaults.tone,repeat:defaults.repeat,
    visual:source.notify===false?false:defaults.visual,system:source.notify===false?false:defaults.system,volume:defaults.volume
  }:null;
  profileForm('edit-',source?.alarm??legacy);
  $('favorite-alarm-details').open=Boolean(source?.alarm||legacy);
  if($('favorite-more-options')) $('favorite-more-options').open=false;
  $('edit-error').textContent='';
  $('favorite-dialog').showModal();
  $('edit-name').focus();
}
$('edit-cancel').addEventListener('click',()=>$('favorite-dialog').close());
$('favorite-dialog').addEventListener('close',()=>{editID=null;$('edit-form').reset();$('edit-error').textContent='';});
$('edit-form').addEventListener('submit',async e=>{
  e.preventDefault();
  $('edit-submit').disabled=true;
  try {
    const name=$('edit-name').value.trim(), duration=$('edit-duration').value.trim();
    if(!name)throw Error('Bitte einen Namen eingeben.');
    if(!duration)throw Error('Bitte eine Dauer eingeben.');
    const normalized=parseQuickTimer(duration);
    if(normalized.name)throw Error('Bitte die Dauer getrennt vom Namen eingeben.');
    const shortcut=$('edit-shortcut').value;
    if(shortcut && (current.templates||[]).some(f=>f.id!==editID && f.shortcut===shortcut))throw Error(`Schnellstart #${shortcut} ist bereits vergeben.`);
    const payload={name,duration:normalized.duration,group:$('edit-group').value.trim(),shortcut,browser_only:STANDALONE||$('edit-browser-only').checked,alarm:profileValues('edit-')};
    await api(editID?'/templates/'+encodeURIComponent(editID):'/templates',editID?'PUT':'POST',payload);
    $('favorite-dialog').close();
    await reload();
    // No toast needed: the newly saved favorite is immediately visible.
    $('fav-title').scrollIntoView({block:'nearest',behavior:window.matchMedia?.('(prefers-reduced-motion: reduce)').matches?'instant':'smooth'});
  } catch(error) {$('edit-error').textContent=error.message;}
  finally {$('edit-submit').disabled=false;}
});
// Use the *visible* viewport, not the layout viewport behind a soft keyboard.
// Avoid reflow by updating only when the dimensions actually change.
function updateDialogViewport() {
  const v=window.visualViewport;
  const style=document.documentElement.style;
  if(v) {
    style.setProperty('--dialog-viewport-height',`${Math.max(0,v.height)}px`);
    style.setProperty('--dialog-viewport-top',`${v.offsetTop}px`);
  } else {
    style.removeProperty('--dialog-viewport-height');
    style.removeProperty('--dialog-viewport-top');
  }
}
window.visualViewport?.addEventListener('resize',updateDialogViewport);
window.visualViewport?.addEventListener('scroll',updateDialogViewport);
window.addEventListener('resize',updateDialogViewport);
updateDialogViewport();
$('open-settings').addEventListener('click',()=>{$('settings-dialog').showModal();updateDialogViewport();});
$('open-shortcuts').addEventListener('click',()=>$('shortcuts-dialog').showModal());
document.querySelectorAll('[data-close-dialog]').forEach(button=>button.addEventListener('click',()=>$(button.dataset.closeDialog).close()));
let pendingConfirmation=null;
function confirmAction({title,description,label='Bestätigen',run,danger=true}) {
  pendingConfirmation=run;
  $('confirm-title').textContent=title;
  $('confirm-description').textContent=description;
  $('confirm-ok').textContent=label;
  $('confirm-ok').classList.toggle('danger-button',danger);
  $('confirm-dialog').showModal();
  $('confirm-cancel').focus();
}
// Only two-action confirmation dialogs use horizontal arrow navigation.
// Editing dialogs must keep the arrows for text cursors, selects and sliders.
$('confirm-dialog').addEventListener('keydown',event=>{
  if(event.defaultPrevented || event.repeat || event.isComposing ||
     event.altKey || event.ctrlKey || event.metaKey) return;
  if(event.key==='ArrowLeft' || event.key==='ArrowRight') {
    event.preventDefault();
    $(event.key==='ArrowLeft'?'confirm-cancel':'confirm-ok').focus();
  }
});
$('confirm-cancel').addEventListener('click',()=>$('confirm-dialog').close());
$('confirm-dialog').addEventListener('close',()=>pendingConfirmation=null);
$('confirm-ok').addEventListener('click',()=>{
  const run=pendingConfirmation;
  $('confirm-dialog').close();
  if(run)Promise.resolve().then(run).catch(err=>message(err.message,true));
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
function confirmStop(id) {
  const timer=(current.timers||[]).find(t=>t.id===id && (t.status==='running'||t.status==='paused'));
  if(!timer)return;
  confirmAction({title:'Timer beenden?',description:L('„{name}“ ({time} verbleibend) wirklich beenden?',{name:timer.name,time:fmt(remaining(timer))}),label:'Timer beenden',
    run:()=>perform(()=>timerControl(id,'stop'))});
}
// Esc works across the main screen, including buttons, favorites and the
// advanced timer options. Open dialogs retain native Escape-to-close.
document.addEventListener('keydown',event=>{
  if(event.defaultPrevented || event.repeat || event.isComposing ||
     event.altKey || event.ctrlKey || event.metaKey || document.querySelector('dialog[open]')) return;
  if(event.key==='Escape') {
    event.preventDefault();
    if(recentOpen){closeRecent();event.preventDefault();return;}
    if(event.target===quickInput && (quickInput.value || quickError.textContent)) {
      resetRecentSelection();
      quickInput.value='';
      quickInput.dispatchEvent(new Event('input',{bubbles:true}));
      return;
    }
    // No special timer focus mode: leaving any focused control makes global
    // arrows available. From the unfocused page Esc returns to quick entry.
    if(event.target===document.body || event.target===document.documentElement) focusQuickInput(true);
    else if(document.activeElement instanceof HTMLElement) document.activeElement.blur();
    return;
  }
  // Keep native caret / range / selector navigation intact. Elsewhere the
  // left/right arrows cycle the displayed timer without a special focus mode.
  if(event.target.closest('input,textarea,select,[contenteditable="true"],[role="slider"],[role="spinbutton"],[role="radiogroup"],[role="listbox"],[role="tablist"]'))return;
  if(event.key==='ArrowLeft' || event.key==='ArrowRight') {
    if(!focusTimer())return;
    event.preventDefault();
    selectTimer(event.key==='ArrowLeft'?-1:1);
    return;
  }
  const key=event.key.toLowerCase();
  if(key==='n') {event.preventDefault();focusQuickInput(true);}
  else if(event.key==='?' || (event.code==='Slash' && event.shiftKey)) {event.preventDefault();$('shortcuts-dialog').showModal();}
  else if(key==='p' && focusTimer()) {
    event.preventDefault();
    const timer=focusTimer();
    manualFocusID=timer.id; // pausing keeps this timer in view for resuming
    perform(()=>timerControl(timer.id,timer.status==='paused'?'resume':'pause'));
  } else if(key==='s' && focusTimer()) {event.preventDefault();confirmStop(focusTimer().id);}
});
document.addEventListener('click',e=>{
  const groupButton=e.target.closest('button[data-group-choice]');
  if(groupButton){
    const prefix=groupButton.dataset.groupChoice;
    $(prefix+'-group').value=groupButton.dataset.groupName;
    highlightGroupPicks(prefix);
    return;
  }
  const button=e.target.closest('button[data-action]');
  if(!button)return;
  const {action,id}=button.dataset;
  if(action==='show-timer') {manualFocusID=id;draw();return;}
  if(action==='group-up'||action==='group-down'){moveGroup(button.dataset.group,action==='group-up'?-1:1);return;}
  if(action==='new-fav-group'){openFavoriteEditor(null,{group:button.dataset.group});return;}
  if(action==='rename-group'){renameGroup=button.dataset.group;$('group-rename').value=renameGroup;$('group-dialog').showModal();$('group-rename').focus();$('group-rename').select();return;}
  if(action==='start-fav'){
    if(!sorting){
      perform(()=>startTimer('','',id)).then(success=>{
        if(success && document.activeElement===button)button.blur();
      });
    }
    return;
  }
  if(action==='edit-fav'){
    const fav=(current.templates||[]).find(t=>t.id===id);if(!fav)return;
    openFavoriteEditor(fav);return;
  }
  if(action==='delete-fav'){
    const fav=(current.templates||[]).find(t=>t.id===id);
    if(fav)confirmAction({title:'Favorit löschen?',description:L('„{name}“ wird dauerhaft gelöscht.',{name:fav.name}),label:'Löschen',run:()=>perform(()=>api('/templates/'+encodeURIComponent(id),'DELETE'))});
    return;
  }
  if(action==='save-as-favorite'){
    const timer=(current.timers||[]).find(t=>t.id===id);
    if(timer)openFavoriteEditor(null,{timer});
    return;
  }
  if(action==='stop') {confirmStop(id);return;}
  if(action==='pause'||action==='resume') {
    if(focusTimer()?.id===id) manualFocusID=id;
  }
  if(action==='adjust'){
    const delta=button.dataset.delta||'+1m';
    perform(()=>timerControl(id,'adjust',delta));return;
  }
  perform(()=>timerControl(id,action));
});
const events=STANDALONE?null:new EventSource('/api/events');
events?.addEventListener('state',e=>{
  try{const event=JSON.parse(e.data);apply(event.state);if(event.type==='finished'&&event.timer)alertFinished(event.timer);$('connect').textContent='Verbunden';$('live').classList.remove('off');}
  catch(error){console.error(error);}
});
events?.addEventListener('error',()=>{$('connect').textContent='Verbinde erneut …';$('live').classList.add('off');});
initBrowserAlarms();
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
localTimers=new LocalTimers(combine,t=>alertFinished(t));
if(STANDALONE){
  standaloneStore=createStandaloneAPI(()=>reload().catch(console.warn));
  $('timer-browser-only').checked=true;
  $('edit-browser-only').checked=true;
}
combine();
// Focus the main entry on first load, unless another control already has focus.
if (document.activeElement === document.body) focusQuickInput();
// PWA registration and user-confirmed updates are managed in /pwa-update.js.
window.setInterval(refreshTime,250);
if(!STANDALONE)window.setInterval(()=>{if(events.readyState!==EventSource.OPEN)reload().catch(()=>{});},10000);
reload().catch(()=>{
  if(!STANDALONE)message('Server nicht erreichbar. Lokale Timer funktionieren offline; gespeicherte Favoriten können lokal gestartet werden.',false);
});
if(STANDALONE){
  $('standalone-export')?.addEventListener('click',()=>{
    const data=JSON.stringify({...standaloneStore.exportBackup(localTimers.all()),recent:recentTimers.all()},null,2);
    const url=URL.createObjectURL(new Blob([data],{type:'application/json'}));
    const a=document.createElement('a');a.href=url;a.download='nexttimer-backup.json';a.click();
    setTimeout(()=>URL.revokeObjectURL(url),1000);
  });
  $('standalone-import')?.addEventListener('change',async e=>{
    const file=e.target.files?.[0];if(!file)return;
    try{
      const raw=JSON.parse(await file.text());
      const data=standaloneStore.importBackup(raw);
      confirmAction({title:'Sicherung wiederherstellen?',description:'Alle vorhandenen lokalen Favoriten und Timer werden durch diese Sicherung ersetzt.',label:'Wiederherstellen',danger:true,run:async()=>{
        standaloneStore.replaceFavorites(data.favorites);
        recentTimers.replace(raw.recent || []);
        localStorage.setItem('nexttimer.local-timers.v1',JSON.stringify(data.timers));
        localTimers.timers=localTimers.read();
        await reload();drawRecent();message('Sicherung importiert.');
      }});
    }catch(err){message(err.message,true);}
    finally{e.target.value='';}
  });
}
