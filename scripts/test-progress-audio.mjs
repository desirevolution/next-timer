// Dependency-free scheduler tests. Sound synthesis itself uses Web Audio.
import assert from 'node:assert/strict';
import fs from 'node:fs';
const source=fs.readFileSync(new URL('../internal/server/ui/js/progress-audio.js',import.meta.url),'utf8');
const {ProgressScheduler,PROGRESS_DEFAULTS,readProgressSettings,saveProgressSettings}=
  await import('data:text/javascript;base64,'+Buffer.from(source).toString('base64'));
const settings=(override={})=>({...PROGRESS_DEFAULTS,milestones:true,ticking:true,...override});
const makeTimer=(id,started,duration,local=false)=>({id,created_at_ms:started,
  duration_ms:duration,deadline_ms:started+duration,remaining_ms:duration,status:'running',local});
const capture=(opts=settings())=>{
  let now=0;const logs=[];
  const engine=new ProgressScheduler({milestone:(i,v)=>logs.push(['milestone',i,now,v]),
    tick:v=>logs.push(['tick',null,now,v]),silence:()=>{}},opts);
  const update=(timers,clock,skew=0,connected=true)=>{now=clock;engine.update(timers,skew,connected,clock);};
  return {engine,logs,update};
};
const events=(logs,type)=>logs.filter(e=>e[0]===type);
{
  const c=capture(),t=makeTimer('full',100_000,60_000);
  for(let ms=0;ms<=60_000;ms+=250)c.update([t],100_000+ms);
  assert.deepEqual(events(c.logs,'milestone').map(x=>x[1]),[0,1,2], '25 / 50 / 75 once each');
  assert.deepEqual(events(c.logs,'milestone').map(x=>x[2]),[115_000,130_000,145_000]);
  assert.equal(events(c.logs,'tick').length,10,'countdown 10 through 1');
  assert.deepEqual(events(c.logs,'tick').map(x=>x[2]),Array.from({length:10},(_,i)=>150_000+i*1000));
}
{
  const c=capture(),t=makeTimer('short',100_000,30_000);
  for(let ms=0;ms<=30_000;ms+=250)c.update([t],100_000+ms);
  assert.equal(events(c.logs,'milestone').length,0,'under default 1 minute: no milestones');
  assert.equal(events(c.logs,'tick').length,10,'30 seconds: ten ticks');
}
{
  const c=capture(),t=makeTimer('tiny',100_000,8_000);
  for(let ms=0;ms<=8_000;ms+=250)c.update([t],100_000+ms);
  assert.equal(c.logs.length,0,'8 seconds: existing end alarm only');
}
{
  const c=capture(),t=makeTimer('exact10',100_000,10_000);
  for(let ms=0;ms<=10_000;ms+=250)c.update([t],100_000+ms);
  assert.equal(events(c.logs,'tick').length,10,'10-second timer ticks 10 to 1');
}
{
  const c=capture(settings({minDuration:120})),t=makeTimer('min2',100_000,60_000);
  for(let ms=0;ms<=60_000;ms+=250)c.update([t],100_000+ms);
  assert.equal(events(c.logs,'milestone').length,0,'custom minimum suppresses only milestones');
  assert.equal(events(c.logs,'tick').length,10);
}
{
  const c=capture(),first=makeTimer('first',100_000,60_000),second=makeTimer('second',100_000,65_000);
  for(let ms=0;ms<=54_000;ms+=250)c.update([first,second],100_000+ms);
  const oldCount=events(c.logs,'tick').length;
  first.status='paused';
  c.update([first,second],154_250); // handoff without extra sound
  assert.equal(events(c.logs,'tick').length,oldCount);
  for(let ms=54_500;ms<=65_000;ms+=250)c.update([first,second],100_000+ms);
  assert(events(c.logs,'tick').length>oldCount,'second timer takes over automatically');
}
{
  const c=capture(),t=makeTimer('hidden',100_000,60_000);
  c.update([t],110_000);
  c.update([t],131_000); // page was suspended for 21 seconds
  assert.equal(c.logs.length,0,'no catch-up from background throttling');
  for(let ms=31_250;ms<=46_000;ms+=250)c.update([t],100_000+ms);
  assert.deepEqual(events(c.logs,'milestone').map(x=>x[1]),[2],'future 75% milestone still works');
}
{
  const c=capture(),t=makeTimer('reconnect',100_000,60_000);
  for(let ms=0;ms<=14_000;ms+=250)c.update([t],100_000+ms);
  for(let ms=14_250;ms<30_000;ms+=250)c.update([t],100_000+ms,0,false);
  c.update([t],130_000,0,true);
  assert.deepEqual(events(c.logs,'milestone').map(x=>x[1]),[],'no missed milestone on reconnect');
  for(let ms=30_250;ms<=46_000;ms+=250)c.update([t],100_000+ms);
  assert.deepEqual(events(c.logs,'milestone').map(x=>x[1]),[2]);
}
{
  const c=capture(),t=makeTimer('toggle',100_000,60_000);
  for(let ms=0;ms<=24_000;ms+=250)c.update([t],100_000+ms);
  c.engine.setSettings(settings({milestones:false,ticking:false}));
  for(let ms=24_250;ms<=36_000;ms+=250)c.update([t],100_000+ms);
  c.engine.setSettings(settings());
  for(let ms=36_250;ms<=46_000;ms+=250)c.update([t],100_000+ms);
  assert.deepEqual(events(c.logs,'milestone').map(x=>x[1]),[0,2],'no replay on enabling');
}
{
  const c=capture(),t=makeTimer('clock',100_000,60_000);
  c.update([t],114_750);
  c.update([t],115_000,3000); // server clock offset suddenly jumps
  assert.equal(c.logs.length,0,'clock correction does not trigger false milestone');
}
{
  const c=capture(),t=makeTimer('local',100_000,60_000,true);
  for(let ms=0;ms<=16_000;ms+=250)c.update([t],100_000+ms,0,false);
  assert.deepEqual(events(c.logs,'milestone').map(x=>x[1]),[0],'local timers run without server');
}
{
  const mem=new Map();const storage={getItem:k=>mem.get(k)??null,setItem:(k,v)=>mem.set(k,v)};
  assert.deepEqual(readProgressSettings(storage),PROGRESS_DEFAULTS);
  saveProgressSettings(settings({minDuration:30,volume:150}),storage);
  assert.equal(readProgressSettings(storage).minDuration,60,'min 60 seconds');
  assert.equal(readProgressSettings(storage).volume,100,'volume clamped');
  saveProgressSettings(settings({minDuration:300,volume:22}),storage);
  assert.equal(readProgressSettings(storage).minDuration,300);
}
console.log('Progress audio: exact thresholds, 10 ticks, short timers, handoff, pause, reconnection, skips, clock jumps, settings OK');
