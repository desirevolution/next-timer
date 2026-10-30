#!/usr/bin/env python3
"""DOM-level quick-start regression test without any network navigation.

The app's real ES modules are combined for this restricted test browser; the
production build still loads the original unmodified modules independently.
"""
from pathlib import Path
import re
from playwright.sync_api import sync_playwright

ROOT=Path(__file__).resolve().parents[1]
UI=ROOT/'dist/static'
MODULES={
    'local':('local-timers.js',['parseTimerDuration','LocalTimers']),
    'sort':('favorite-sort.js',['enableFavoriteSorting']),
    'alerts':('alerts.js',['alertFinished','initBrowserAlarms','getDefaults','saveDefaults','permission','requestNotifications','unlockAudio','playFinishSound']),
    'standalone':('standalone-store.js',['createStandaloneAPI']),
    'quick':('quick-timer.js',['parseQuickTimer']),
}

def browser_bundle():
    chunks=[]
    for module,(file,exports) in MODULES.items():
        source=(UI/'js'/file).read_text().replace('import.meta.url', '"https://nexttimer.example.test/js/alerts.js"')
        source=re.sub(r"^import\s+\{[^}]*\}\s+from\s+['\"][^'\"]+['\"];?\s*$",'',source,flags=re.M)
        source=re.sub(r'\bexport\s+(?=(?:async\s+)?(?:function|class|const)\b)','',source)
        pre='const {parseTimerDuration}=module_local;' if module in ['standalone','quick'] else ''
        chunks.append(f'const module_{module}=(()=>{{{pre}\n{source}\nreturn {{{",".join(exports)}}};}})();')
    app=(UI/'js/app.js').read_text()
    mapping={'favorite-sort':'sort','standalone-store':'standalone','local-timers':'local','quick-timer':'quick','alerts':'alerts'}
    for filename,variable in mapping.items():
        pattern=rf'^import\s+\{{([^}}]+)\}}\s+from\s+[\'\"]\./{filename}\.js\?v=[^\'\"]+[\'\"];?\s*$'
        app,count=re.subn(pattern,lambda m:f'const {{{m.group(1)}}}=module_{variable};',app,count=1,flags=re.M)
        assert count==1,(filename,app[:700])
    return '\n'.join(chunks)+'\n'+app

html=(UI/'index.html').read_text().replace('<html lang="de" data-mode="standalone">','<html lang="de" data-mode="standalone">')
html=re.sub(r'<script[^>]+src="[^"]+"[^>]*></script>','',html)
html=re.sub(r'<link[^>]+(?:rel="(?:stylesheet|manifest|icon)")[^>]*>','',html)
html=html.replace('</head>',f'<style>{(UI/"styles.css").read_text()}</style></head>')

with sync_playwright() as pw:
    browser=pw.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox'])
    for width in (1100,375):
        page=browser.new_page(viewport={'width':width,'height':850},is_mobile=width==375,has_touch=width==375)
        errors=[]
        page.on('pageerror',lambda error:errors.append(str(error)))
        page.set_content(html)
        page.evaluate("""() => {const d=new Map();Object.defineProperty(window,'localStorage',{value:{getItem:k=>d.has(k)?d.get(k):null,setItem:(k,v)=>d.set(k,''+v),removeItem:k=>d.delete(k),clear:()=>d.clear()}});}""")
        page.add_script_tag(content=browser_bundle())
        page.wait_for_function("document.querySelector('#quick-input') && document.querySelector('#standalone-export')")
        assert page.locator('.connection').count()==0
        field=page.locator('#quick-input')
        assert field.input_value()==''
        assert page.locator('#quick-start').is_disabled()
        assert not page.locator('.shortcut-hint').count()
        # Physical-keyboard desktops start ready to type; touch devices never
        # trigger an unwanted software keyboard on initial page load.
        assert field.evaluate('(el)=>el===document.activeElement')==(width==1100)
        page.locator('#sort-toggle').focus()
        page.keyboard.press('n')
        assert field.evaluate('(el)=>el===document.activeElement'), 'N always focuses entry when explicitly requested'
        field.fill('bad input');field.press('Enter')
        assert page.locator('#quick-error').inner_text()
        assert page.locator('#hero-name').inner_text()=='Noch kein Timer aktiv'
        assert page.locator('#focus-panel').evaluate('el=>!el.hasAttribute("tabindex") && !el.inert')
        field.press('Escape')
        assert field.input_value()==''
        assert page.locator('#quick-error').inner_text()==''
        page.locator('[data-quick-duration="3m"]').click()
        assert field.input_value()=='3m'
        assert page.locator('#hero-name').inner_text()=='Noch kein Timer aktiv'
        field.fill('1m Tee');field.press('Enter')
        page.wait_for_function("document.querySelector('#hero-name')?.textContent==='Tee'")
        assert field.input_value()==''
        page.wait_for_function("document.activeElement !== document.querySelector('#quick-input')")
        assert page.locator('#quick-start').is_disabled()
        field.fill('2m Kaffee');page.locator('#quick-start').click()
        page.wait_for_function("document.querySelector('#timer-count')?.textContent==='1'")
        assert page.locator('#hero-name').inner_text()=='Kaffee'
        assert 'Tee' in page.locator('#timers').inner_text()
        assert not field.evaluate('(el)=>el===document.activeElement'), 'Start must release text focus for global shortcuts'
        # Starting a favorite does not reactivate the text input; shortcuts work right away.
        page.locator('#new-favorite').click()
        page.locator('#edit-name').fill('Kurz')
        page.locator('#edit-duration').fill('1m')
        page.locator('#edit-submit').click()
        page.wait_for_function("document.querySelectorAll('.fav-start').length===1")
        favorite=page.locator('.fav-start')
        favorite.click()
        page.wait_for_function("document.querySelector('#timer-count')?.textContent==='2'")
        assert not field.evaluate('(el)=>el===document.activeElement'), 'Pointer favorite should not reactivate the entry'
        favorite.focus()
        favorite.press('Enter')
        page.wait_for_function("document.querySelector('#timer-count')?.textContent==='3'")
        assert favorite.evaluate('(el)=>el===document.activeElement'), 'Keyboard favorite must keep its focus'
        assert page.evaluate('document.documentElement.scrollWidth <= innerWidth')
        assert not errors,errors
        print(f'Quick Timer DOM {width}px: desktop/touch focus, N, Enter, Esc, chips, favorites, no overflow / JS errors OK')
        page.close()
    browser.close()

# The same UI uses the server API (rather than local timers) in hosted mode.
server_html=(ROOT/'internal/server/ui/index.html').read_text()
server_html=re.sub(r'<script[^>]+src="[^"]+"[^>]*></script>','',server_html)
server_html=re.sub(r'<link[^>]+(?:rel="(?:stylesheet|manifest|icon)")[^>]*>','',server_html)
with sync_playwright() as pw:
    browser=pw.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox'])
    page=browser.new_page();errors=[]
    page.on('pageerror',lambda e:errors.append(str(e)))
    page.set_content(server_html)
    page.evaluate("""() => {
      const d=new Map();Object.defineProperty(window,'localStorage',{value:{getItem:k=>d.get(k)??null,setItem:(k,v)=>d.set(k,''+v)}});
      window.quickRequests=[];window.serverTimers=[];
      window.EventSource=class{constructor(){this.readyState=1;}addEventListener(){} static OPEN=1};
      window.fetch=(url,options={})=>{
        const reply=(body,status=200)=>Promise.resolve(new Response(JSON.stringify(body),{status,headers:{'Content-Type':'application/json'}}));
        if(url==='/api/state')return reply({timers:window.serverTimers,templates:[],groups:[],server_now_ms:Date.now()});
        if(url==='/api/timers'&&options.method==='POST'){
          const req=JSON.parse(options.body);window.quickRequests.push(req);
          const now=Date.now(),ms=180000;
          window.serverTimers.push({id:'t1',name:req.name,duration_ms:ms,remaining_ms:ms,deadline_ms:now+ms,created_at_ms:now,status:'running'});
          return reply(window.serverTimers[0],201);
        }
        return Promise.reject(Error('Unexpected request '+url));
      };
    }""")
    page.add_script_tag(content=browser_bundle())
    page.locator('#quick-input').fill('3m Tee')
    page.locator('#quick-input').press('Enter')
    page.wait_for_function('window.quickRequests.length===1')
    assert page.evaluate('window.quickRequests[0]')=={'duration':'3m','name':'Tee','browser_only':False,'alarm':None}
    page.wait_for_function("document.querySelector('#quick-input').value===''")
    assert not errors,errors
    print('Quick Timer server mode: POST /api/timers, server defaults, no local-only flag OK')
    browser.close()
