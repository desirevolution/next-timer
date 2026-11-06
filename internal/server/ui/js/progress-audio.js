// Optional, per-browser progress audio. No server changes, network requests or
// audio files; independent of the existing end-of-timer browser alarm.
const STORAGE_KEY = 'nexttimer.progress-audio.v1';
export const PROGRESS_DEFAULTS = Object.freeze({milestones:false,ticking:false,volume:35,minDuration:60});
export const MIN_DURATION_OPTIONS = Object.freeze([60,120,300,600]);
const MILESTONES = [0.25,0.50,0.75];
const MAX_OBSERVATION_GAP_MS = 1250; // Never replay missed events after tab throttling.

export function readProgressSettings(storage=globalThis.localStorage) {
  try {
    const stored=JSON.parse(storage.getItem(STORAGE_KEY)||'null');
    if (!stored || typeof stored!=='object') return {...PROGRESS_DEFAULTS};
    return {
      milestones:stored.milestones===true,
      ticking:stored.ticking===true,
      volume:Number.isFinite(stored.volume)?Math.min(100,Math.max(0,Math.round(stored.volume))):PROGRESS_DEFAULTS.volume,
      minDuration:MIN_DURATION_OPTIONS.includes(stored.minDuration)?stored.minDuration:PROGRESS_DEFAULTS.minDuration
    };
  } catch {return {...PROGRESS_DEFAULTS};}
}
export function saveProgressSettings(settings,storage=globalThis.localStorage) {
  const clean={
    milestones:settings.milestones===true,
    ticking:settings.ticking===true,
    volume:Number.isFinite(settings.volume)?Math.min(100,Math.max(0,Math.round(settings.volume))):PROGRESS_DEFAULTS.volume,
    minDuration:MIN_DURATION_OPTIONS.includes(settings.minDuration)?settings.minDuration:PROGRESS_DEFAULTS.minDuration
  };
  storage.setItem(STORAGE_KEY,JSON.stringify(clean));
  return clean;
}

// Pure scheduling logic is independently testable without Web Audio or a DOM.
// The server clock offset is applied only to server timers; local timers use the
// browser clock. Playing sound does not depend on which timer is displayed.
export class ProgressScheduler {
  constructor(play,settings=PROGRESS_DEFAULTS) {
    this.play=play;
    this.settings={...settings};
    this.previous=null;
    this.played=new Map();
  }
  setSettings(settings) {
    const changed = settings.milestones!==this.settings.milestones ||
      settings.ticking!==this.settings.ticking ||
      settings.minDuration!==this.settings.minDuration;
    this.settings={...settings};
    // The next observation is a fresh baseline, never an excuse to replay a
    // milestone that was crossed while the setting was disabled.
    if(changed)this.previous=null;
    if(!settings.milestones&&!settings.ticking)this.previous=null;
  }
  update(timers=[],serverSkew=0,serverConnected=true,now=Date.now()) {
    const active=timers.filter(t=>t.status==='running' && (serverConnected || t.local) &&
      Number.isFinite(t.deadline_ms) && Number.isFinite(t.duration_ms) && t.duration_ms>=1000)
      .map(t=>({...t, remaining:t.deadline_ms-now-(t.local?0:serverSkew)}))
      .filter(t=>t.remaining>0)
      .sort((a,b)=>a.remaining-b.remaining||a.created_at_ms-b.created_at_ms||String(a.id).localeCompare(String(b.id)));
    const leader=active[0];
    const ids=new Set(timers.filter(t=>t.status==='running'||t.status==='paused')
      .map(t=>`${t.local?'local':'server'}:${t.id}:${t.created_at_ms}`));
    for(const key of this.played.keys())if(!ids.has(key))this.played.delete(key);
    if(!leader || (!this.settings.milestones&&!this.settings.ticking)) {
      if(this.previous)this.play.silence?.();
      this.previous=null;
      return;
    }
    const key=`${leader.local?'local':'server'}:${leader.id}:${leader.created_at_ms}`;
    const sample={key,remaining:leader.remaining,deadline:leader.deadline_ms,
      duration:leader.duration_ms,at:now};
    const prev=this.previous;
    this.previous=sample;
    if(!prev || prev.key!==key) {
      if(prev && prev.key!==key)this.play.silence?.();
      // A newly started 10-second timer has no prior observation crossing
      // the 10-second boundary. Begin with one initial '10' tick only if
      // it was truly just created, not after reload or leadership transfer.
      const created=leader.created_at_ms-(leader.local?0:serverSkew);
      if(this.settings.ticking && leader.duration_ms===10000 &&
         now-created>=0 && now-created<500 && leader.remaining>9500 && leader.remaining<=10000)
        this.play.tick(this.settings.volume/100);
      return;
    }
    // Reconnects, wall-clock jumps and server time-skew corrections must not
    // masquerade as legitimately crossed progress thresholds.
    const expectedRemaining=prev.remaining-(now-prev.at);
    if(prev.deadline!==sample.deadline || prev.duration!==sample.duration ||
       now-prev.at>MAX_OBSERVATION_GAP_MS || now<=prev.at || leader.remaining>=prev.remaining ||
       Math.abs(expectedRemaining-leader.remaining)>450)return;

    // For a normal 60-second timer these marks occur at 45s, 30s and 15s
    // remaining. Milestone notes never interrupt the last-10-second ticks.
    if(this.settings.milestones && leader.duration_ms>=this.settings.minDuration*1000) {
      const heard=this.played.get(key)||new Set();
      for(let i=0;i<MILESTONES.length;i++) {
        const boundary=leader.duration_ms*(1-MILESTONES[i]);
        if(prev.remaining>boundary && leader.remaining<=boundary && leader.remaining>10000 && !heard.has(i)) {
          heard.add(i);
          this.play.milestone(i,this.settings.volume/100);
        }
      }
      this.played.set(key,heard);
    }
    // ceil() names the *current* displayed second (10, 9, ... 1).
    // Never generate multiple ticks for skipped seconds / background throttling.
    if(this.settings.ticking && leader.duration_ms>=10000) {
      const previousSecond=Math.ceil(prev.remaining/1000);
      const second=Math.ceil(leader.remaining/1000);
      if(second>=1 && second<=10 && second<previousSecond)
        this.play.tick(this.settings.volume/100);
    }
  }
}

// Browser sound generator. Reuses neither the end-alarm tone nor its volume;
// stopping one kind of audio cannot silence the other.
export class ProgressSound {
  constructor(onStatus=()=>{}) {
    this.context=null;
    this.nodes=new Set();
    this.status='idle';
    this.onStatus=onStatus;
  }
  setStatus(status) {
    if(this.status===status)return;
    this.status=status;
    this.onStatus(status);
  }
  unlock() {
    const AudioContextClass=globalThis.AudioContext||globalThis.webkitAudioContext;
    if(!AudioContextClass) {this.setStatus('unavailable');return Promise.resolve();}
    try {
      this.context??=new AudioContextClass();
      if(this.context.state==='running') {this.setStatus('ready');return Promise.resolve();}
      // Browsers may deny resume() without a user gesture (e.g. after reload).
      // Do not consider an unresolved promise as successful permission.
      this.setStatus('blocked');
      return this.context.resume().then(()=>{
        this.setStatus(this.context.state==='running'?'ready':'blocked');
      }).catch(()=>this.setStatus('blocked'));
    } catch {this.setStatus('unavailable');return Promise.resolve();}
  }
  stop() {
    if(!this.context)return;
    for(const {gain,source} of this.nodes) {
      try {gain.gain.cancelScheduledValues(this.context.currentTime);
        gain.gain.setValueAtTime(0,this.context.currentTime);source.stop(this.context.currentTime+.005);} catch {}
    }
    this.nodes.clear();
  }
  makeSource(source,gain) {
    const item={source,gain};this.nodes.add(item);
    source.onended=()=>{this.nodes.delete(item);source.disconnect();gain.disconnect();};
  }
  milestone(index,volume,start=null) {
    if(volume<=0)return;
    this.unlock();
    const ctx=this.context;
    if(ctx?.state!=='running')return;
    const time=start??ctx.currentTime+.012;
    const osc=ctx.createOscillator(),gain=ctx.createGain();
    osc.type='sine';osc.frequency.setValueAtTime([523.25,659.25,783.99][index]||523.25,time);
    gain.gain.setValueAtTime(0,time);
    gain.gain.linearRampToValueAtTime(volume*.115,time+.025);
    gain.gain.exponentialRampToValueAtTime(.0001,time+.21);
    osc.connect(gain);gain.connect(ctx.destination);
    this.makeSource(osc,gain);
    osc.start(time);osc.stop(time+.23);
  }
  tick(volume,start=null) {
    if(volume<=0)return;
    this.unlock();
    const ctx=this.context;
    if(ctx?.state!=='running')return;
    const time=start??ctx.currentTime+.012;
    // Soft, short, filtered noise: a dry click rather than an electronic beep.
    const frames=Math.ceil(ctx.sampleRate*.012);
    const buffer=ctx.createBuffer(1,frames,ctx.sampleRate);
    const samples=buffer.getChannelData(0);
    let seed=0x1257;
    for(let i=0;i<frames;i++) {
      seed=(seed*1664525+1013904223)>>>0;
      samples[i]=((seed/0x100000000)*2-1)*Math.pow(1-i/frames,2);
    }
    const source=ctx.createBufferSource(),filter=ctx.createBiquadFilter(),gain=ctx.createGain();
    source.buffer=buffer;filter.type='bandpass';filter.frequency.value=1600;filter.Q.value=.75;
    gain.gain.setValueAtTime(volume*.12,time);
    source.connect(filter);filter.connect(gain);gain.connect(ctx.destination);
    this.makeSource(source,gain);
    source.start(time);source.stop(time+.014);
    source.addEventListener('ended',()=>filter.disconnect(),{once:true});
  }
  async preview(kind,volume) {
    await this.unlock();
    if(this.context?.state!=='running')return;
    this.stop();
    const base=this.context.currentTime+.05;
    if(kind==='milestones')for(let i=0;i<3;i++)this.milestone(i,volume,base+i*.32);
    else for(let i=0;i<3;i++)this.tick(volume,base+i);
  }
}
