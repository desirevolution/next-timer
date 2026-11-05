#!/usr/bin/env python3
"""End-to-end standalone/browser repeat feature, using real UI modules."""
from pathlib import Path
import json
from playwright.sync_api import sync_playwright
root=Path(__file__).resolve().parents[1]
quick=root/'scripts/test-quick-start-browser.py'
ns={'__file__':str(quick)}
exec(quick.read_text().split('with sync_playwright() as pw:')[0],ns)
with sync_playwright() as pw:
    browser=pw.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox'])
    for width in (1100,375):
        page=browser.new_page(viewport={'width':width,'height':840},is_mobile=width<500,has_touch=width<500)
        errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
        page.set_content(ns['html'])
        page.evaluate('''() => {const d=new Map();Object.defineProperty(window,'localStorage',{value:{getItem:k=>d.get(k)??null,setItem:(k,v)=>d.set(k,''+v),removeItem:k=>d.delete(k)}})}''')
        page.add_script_tag(content=(root/'internal/server/ui/js/i18n.js').read_text())
        page.add_script_tag(content=ns['browser_bundle']())
        page.wait_for_function("document.querySelector('#repeat-last')")
        assert page.locator('#repeat-idle').is_hidden()
        field=page.locator('#quick-input')
        field.fill('1s Espresso');field.press('Enter')
        page.wait_for_function("document.querySelector('#hero-name')?.textContent==='Espresso'")
        page.wait_for_function("document.querySelector('#repeat-idle').hidden===false",timeout=5000)
        record=page.evaluate("JSON.parse(localStorage.getItem('nexttimer.local-last-finished.v1'))")
        assert record['name']=='Espresso' and record['duration']=='1000ms'
        assert 'Espresso' in page.locator('#repeat-last').inner_text()
        assert page.locator('#repeat-last').is_visible()
        page.evaluate("""() => {let l=document.querySelector('#language-select');l.value='en';l.dispatchEvent(new Event('change',{bubbles:true}));}""")
        page.wait_for_function("document.querySelector('#repeat-last-label').textContent.startsWith('Repeat')")
        page.evaluate("""() => {let l=document.querySelector('#language-select');l.value='de';l.dispatchEvent(new Event('change',{bubbles:true}));}""")
        page.wait_for_function("document.querySelector('#repeat-last-label').textContent.endsWith('wiederholen')")
        page.locator('#repeat-last').click()
        page.wait_for_function("document.querySelector('#hero-name')?.textContent==='Espresso' && document.querySelector('#hero-time')?.textContent!=='--:--'")
        assert page.locator('#repeat-idle').is_hidden()
        assert page.evaluate('document.activeElement===document.body')
        initial=page.evaluate("JSON.parse(localStorage.getItem('nexttimer.local-timers.v1')).length")
        page.keyboard.press('r')
        page.wait_for_function(f"JSON.parse(localStorage.getItem('nexttimer.local-timers.v1')).length>{initial}")
        record2=page.evaluate("JSON.parse(localStorage.getItem('nexttimer.local-last-finished.v1'))")
        assert record2==record, 'starting repeat should not replace last completion'
        field.focus();prior=page.evaluate("JSON.parse(localStorage.getItem('nexttimer.local-timers.v1')).length")
        page.keyboard.type('r');assert field.input_value()=='r'
        assert page.evaluate("JSON.parse(localStorage.getItem('nexttimer.local-timers.v1')).length")==prior
        field.fill('')
        # End a timer manually while multiple run; this becomes latest completion.
        page.locator('#hero-stop').click()
        page.locator('#confirm-ok').click()
        page.wait_for_function("JSON.parse(localStorage.getItem('nexttimer.local-last-finished.v1')).ended_at_ms > 0")
        assert not errors,errors
        assert page.evaluate('document.documentElement.scrollWidth<=innerWidth')
        page.locator('#open-settings').click()
        with page.expect_download() as download:
            page.locator('#standalone-export').click()
        backup=json.loads(Path(download.value.path()).read_text())
        assert backup['last_finished']['name']=='Espresso',backup.get('last_finished')
        page.locator('[data-close-dialog="settings-dialog"]').first.click()
        print(f'Local repeat: {width}px, expiration / click / R / text-field protection / stop: OK')
        page.close()
    # Server mode must call the server-wide atomic repeat endpoint, not
    # reconstruct a client-side timer from its local start history.
    page=browser.new_page()
    from re import sub
    html=(root/'internal/server/ui/index.html').read_text()
    html=sub(r'<script[^>]+src="[^"]+"[^>]*></script>','',html)
    html=sub(r'<link[^>]+(?:rel="(?:stylesheet|manifest|icon)")[^>]*>','',html)
    page.set_content(html)
    page.evaluate('''() => {
      const d=new Map();Object.defineProperty(window,'localStorage',{value:{getItem:k=>d.get(k)??null,setItem:(k,v)=>d.set(k,''+v)}});
      window.EventSource=class{constructor(){this.readyState=1;}addEventListener(){} static OPEN=1};
      window.calls=[];
      window.now=Date.now();
      window.serverState={server_now_ms:window.now,timers:[],templates:[],groups:[],
        last_finished:{name:'Server-Kaffee',duration:'2m0s',sound:true,notify:false,ended_at_ms:window.now}};
      window.fetch=(url,opts={})=>{
        const send=(body,status=200)=>Promise.resolve(new Response(JSON.stringify(body),{status,headers:{'Content-Type':'application/json'}}));
        if(url==='/api/state')return send(window.serverState);
        if(url==='/api/repeat'&&opts.method==='POST'){
          window.calls.push('repeat');const t={id:'server-1',name:'Server-Kaffee',duration_ms:120000,
            remaining_ms:120000,deadline_ms:Date.now()+120000,status:'running',created_at_ms:Date.now()};
          window.serverState.timers=[t];return send(t,201);
        }
        throw Error('Unmocked '+url);
      };
    }''')
    page.add_script_tag(content=ns['browser_bundle']())
    page.wait_for_function("document.querySelector('#repeat-idle').hidden===false")
    page.locator('#repeat-last').click()
    page.wait_for_function("window.calls.length===1")
    page.wait_for_function("document.querySelector('#hero-name').textContent==='Server-Kaffee'")
    assert page.locator('#repeat-idle').is_hidden()
    print('Server repeat browser: POST /api/repeat with shared completion state: OK')
    browser.close()
