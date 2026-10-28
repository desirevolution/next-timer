// Small service-worker regression test without browser dependencies.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const ORIGIN = 'https://nexttimer.example';
const script = fs.readFileSync(require('node:path').join(__dirname, '../internal/server/ui/sw.js'), 'utf8');

function harness() {
  const listeners = {};
  const storage = new Map();
  let offline = false;
  let invalidScript = false;
  let skipWaiting = false;
  const url = path => new URL(typeof path === 'string' ? path : path.url, ORIGIN).href;
  class Request {
    constructor(path, options={}) {this.url=url(path);this.mode=options.mode||'cors';this.method='GET';}
  }
  const contentType = path => path.endsWith('.css') ? 'text/css' :
    path.endsWith('.js') ? 'text/javascript' : path.endsWith('.png') ? 'image/png' :
    path.endsWith('.webmanifest') ? 'application/manifest+json' : 'text/html';
  const onlineFetch = async req => {
    if (offline) throw new Error('Network offline');
    const path=new URL(url(req)).pathname;
    const type=invalidScript && path.endsWith('.js') ? 'text/html' : contentType(path);
    return new Response('LIVE '+path, {headers:{'Content-Type':type}});
  };
  const caches = {
    async open(name) {
      if (!storage.has(name)) storage.set(name, new Map());
      return {
        async match(req) {return storage.get(name).get(url(req))?.clone();},
        async put(req, resp) {storage.get(name).set(url(req), resp.clone());},
      };
    },
    async keys() {return [...storage.keys()];},
    async delete(name) {return storage.delete(name);},
  };
  const self = {
    location: {origin: ORIGIN},
    clients: {async claim(){}},
    addEventListener: (type, handler) => listeners[type] = handler,
    skipWaiting: () => {skipWaiting = true;},
  };
  vm.runInNewContext(script, {self, caches, Request, Response, URL, fetch:onlineFetch}, {filename:'sw.js'});
  const lifetime = async type => {
    let result;
    listeners[type]({waitUntil: promise => result=promise});
    await result;
  };
  const request = async (path,mode='cors') => {
    let answer;
    const req = new Request(path,{mode});
    const event = {request:req,respondWith: promise => answer=promise};
    listeners.fetch(event);
    return answer ? await answer : null;
  };
  return {lifetime,request,caches,storage,listeners,setOffline:v=>offline=v,setBadScript:v=>invalidScript=v,wasSkipped:()=>skipWaiting};
}
(async () => {
  const sw = harness();
  await sw.caches.open('nexttimer-pwa-v22');
  await sw.lifetime('install');
  assert((await sw.caches.keys()).includes('nexttimer-pwa-v23'));
  assert.equal(sw.wasSkipped(),false,'updates must wait for user approval');
  await sw.lifetime('activate');
  assert.deepEqual(await sw.caches.keys(),['nexttimer-pwa-v23']);
  assert.match(await (await sw.request('/','navigate')).text(),/^LIVE \/$/,'navigation must be network-first');
  sw.setOffline(true);
  assert.match(await (await sw.request('/','navigate')).text(),/^LIVE \/index.html$/,'offline shell must work');
  assert.match(await (await sw.request('/js/app.js?v=23')).text(),/^LIVE \/js\/app.js$/,'offline JS must work');
  assert.equal(await sw.request('/api/state'),null,'API must not be cached');
  assert.equal(await sw.request('/api/pwa-repair','navigate'),null,'rescue route must bypass old workers');
  assert.equal(await sw.request('/sw.js'),null,'worker script must not be cached');
  assert.equal((await sw.request('/missing.js')).status,503,'missing offline resources should not reject');
  sw.listeners.message({data:{type:'SKIP_WAITING'}});
  assert(sw.wasSkipped(),'activation should happen after approval');
  const broken = harness();
  broken.setBadScript(true);
  await assert.rejects(broken.lifetime('install'), /Invalid offline asset/,'HTML must never be cached as JavaScript');
  console.log('PWA worker: online, offline, update approval, repair, invalid content types OK');
})().catch(err => {console.error(err);process.exitCode=1;});
