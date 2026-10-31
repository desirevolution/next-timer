// Browser-local, durable recent timer configurations. Never persists active timers.
const STORAGE_KEY='nexttimer.recent-timers.v1';
const MAX=5;
function normalize(entry){
  if(!entry||typeof entry!=='object'||typeof entry.duration!=='string'||
     !/^(?:\d+(?:\.\d+)?(?:ms|h|m|s))+$/i.test(entry.duration)||
     typeof entry.name!=='string'||entry.name.length>80)return null;
  const o=entry.options||{};
  if(o.alarm!==undefined&&o.alarm!==null&&typeof o.alarm!=='object')return null;
  return {duration:entry.duration,name:entry.name,at:Number(entry.at)||Date.now(),options:{
    browser_only:!!o.browser_only,notify:o.notify!==false,sound:o.sound!==false,
    alarm:o.alarm?JSON.parse(JSON.stringify(o.alarm)):null
  }};
}
function key(entry){return JSON.stringify([entry.duration,entry.name,entry.options]);}
export class RecentTimers {
  constructor(storage=globalThis.localStorage){
    this.storage=storage;
    let raw=[];
    try{raw=JSON.parse(storage.getItem(STORAGE_KEY)||'[]');}catch{}
    this.entries=[];
    if(Array.isArray(raw))for(const item of raw){
      const entry=normalize(item);
      if(entry&&!this.entries.some(old=>key(old)===key(entry)))this.entries.push(entry);
      if(this.entries.length===MAX)break;
    }
  }
  all(){return this.entries.map(item=>structuredClone(item));}
  add(value){
    const entry=normalize({...value,at:Date.now()});if(!entry)return;
    this.entries=[entry,...this.entries.filter(old=>key(old)!==key(entry))].slice(0,MAX);
    this.save();
  }
  replace(values){
    const clean=[];
    if(Array.isArray(values))for(const item of values){
      const entry=normalize(item);
      if(entry&&!clean.some(old=>key(old)===key(entry)))clean.push(entry);
      if(clean.length===MAX)break;
    }
    this.entries=clean;
    this.save();
  }
  clear(){this.entries=[];this.save();}
  save(){try{this.storage.setItem(STORAGE_KEY,JSON.stringify(this.entries));}catch{}}
}
