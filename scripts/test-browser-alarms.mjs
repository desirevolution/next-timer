import assert from 'node:assert/strict';
const storage=new Map();
let now=100000;
Date.now=()=>now;
const periodic=[];
const elements=new Map();
const el=id=>{
  if(!elements.has(id))elements.set(id,{hidden:false,textContent:'',innerHTML:'',addEventListener(){},dataset:{}});
  return elements.get(id);
};
const freqs=[], volumes=[];
class AudioContextMock {
  state='running'; currentTime=0; destination={};
  createOscillator(){return {type:'',frequency:{setValueAtTime(n){freqs.push(n);}},connect(){},start(){},stop(){}};}
  createGain(){return {gain:{setValueAtTime(){},exponentialRampToValueAtTime(n){volumes.push(n);},cancelScheduledValues(){}},connect(){}};}
}
globalThis.localStorage={getItem:k=>storage.get(k)??null,setItem:(k,v)=>storage.set(k,String(v))};
globalThis.window={isSecureContext:false,AudioContext:AudioContextMock,setInterval:fn=>periodic.push(fn),setTimeout(){},addEventListener(){}};
globalThis.document={getElementById:el,addEventListener(){}};
const {getDefaults,saveDefaults,resolveAlarm,alarmWindowMs,initBrowserAlarms,alertFinished,acknowledgeAlarm,acknowledgeAll}=
  await import('../internal/server/ui/js/alerts.js');
assert.deepEqual(getDefaults(),{tone:'bell',repeat:'once',visual:true,system:true,volume:.6});
saveDefaults({tone:'gong',repeat:'three',visual:false,system:true,volume:.35});
assert.equal(resolveAlarm({}).tone,'gong');
assert.equal(resolveAlarm({sound:false}).tone,'none');
assert.equal(resolveAlarm({notify:false}).visual,false);
assert.equal(resolveAlarm({alarm:{tone:'digital',repeat:'until_ack',visual:true,system:false}}).volume,.35,'legacy profiles inherit global level');
assert.equal(resolveAlarm({alarm:{tone:'digital',repeat:'once',visual:true,system:false,volume:.2}}).volume,.2);
assert.equal(resolveAlarm({alarm:{tone:'digital',repeat:'once',visual:true,system:false,volume:0}}).volume,0,'zero is muted');
assert.equal(alarmWindowMs({tone:'bell',repeat:'once'}),2000);
assert.equal(alarmWindowMs({tone:'bell',repeat:'three'}),8000);
assert.equal(alarmWindowMs({tone:'none',repeat:'until_ack'}),Infinity);
const changes=[];
initBrowserAlarms(pending=>changes.push(pending.length));
const timer=(id,alarm)=>({id,local:true,created_at_ms:123,name:'Timer '+id,alarm});
const persistent={tone:'none',repeat:'until_ack',visual:false,system:false};
alertFinished(timer('a',persistent));
assert.equal(el('alarm-banner').hidden,false,'until_ack must remain visible even when visual alerts disabled');
assert.match(el('alarm-list').innerHTML,/Bestätigen/);
assert.equal(JSON.parse(storage.get('nexttimer.pending-browser-alarms.v1')).length,1);
alertFinished(timer('b',persistent));
assert.match(el('alarm-count').textContent,/2/);
acknowledgeAlarm('local:a:123');
assert.equal(JSON.parse(storage.get('nexttimer.pending-browser-alarms.v1')).length,1);
acknowledgeAll();
assert.equal(el('alarm-banner').hidden,true);
assert.equal(JSON.parse(storage.get('nexttimer.pending-browser-alarms.v1')).length,0);
// Non-repeating red ends after the same brief window as the audible pattern.
const once={tone:'none',repeat:'once',visual:true,system:false};
alertFinished(timer('c',once));
assert.equal(el('alarm-banner').hidden,false);
assert.match(el('alarm-list').innerHTML,/alarm-pulsing/);
now+=1999;periodic.forEach(fn=>fn());
assert.equal(el('alarm-banner').hidden,false);
now+=1;periodic.forEach(fn=>fn());
assert.equal(el('alarm-banner').hidden,true);
// Three red cycles last until the last sound/visual cycle has ended.
const three={tone:'none',repeat:'three',visual:true,system:false};
alertFinished(timer('d',three));
now+=7900;periodic.forEach(fn=>fn());
assert.equal(el('alarm-banner').hidden,false);
now+=100;periodic.forEach(fn=>fn());
assert.equal(el('alarm-banner').hidden,true);
// Exactly the selected digital profile sounds first: no extra legacy bell/gong.
const digital={tone:'digital',repeat:'once',visual:true,system:false,volume:.2};
const before=freqs.length;
alertFinished(timer('e',digital));
assert.deepEqual(freqs.slice(before),[784,784,1046]);
assert.ok(volumes.some(v=>v>0.01&&v<0.02),'individual gain should use configured volume');
const mutedBefore=freqs.length;
alertFinished(timer('f',{...digital,volume:0}));
assert.equal(freqs.length,mutedBefore,'muted profile should not play a sound');
now+=2000;periodic.forEach(fn=>fn());
assert.equal(el('alarm-banner').hidden,true);
// System notifications must not trigger an additional OS default sound.
const nativeNotifications=[];
window.isSecureContext=true;
globalThis.Notification=class {static permission='granted';constructor(title,options){nativeNotifications.push({title,options});}};
window.Notification=globalThis.Notification;
alertFinished(timer('g',{tone:'none',repeat:'once',visual:false,system:true}));
await Promise.resolve();
assert.equal(nativeNotifications.length,1);
assert.equal(nativeNotifications[0].options.silent,true);
console.log('Browser alarms: only selected initial tone, volume override, synced visual duration, repeats and acknowledgements OK');
