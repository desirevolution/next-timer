#!/usr/bin/env python3
"""Real DOM/settings/backup integration for progress audio, desktop and mobile."""
from pathlib import Path
from playwright.sync_api import sync_playwright
import json
root=Path(__file__).resolve().parents[1]
quick=root/'scripts/test-quick-start-browser.py'
ns={'__file__':str(quick)}
exec(quick.read_text().split('with sync_playwright() as pw:')[0],ns)
with sync_playwright() as pw:
    browser=pw.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox'])
    for width in (1100,375):
        page=browser.new_page(viewport={'width':width,'height':800},is_mobile=width<500,has_touch=width<500,accept_downloads=True)
        errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
        page.set_content(ns['html'])
        page.evaluate('''() => {
          const data=new Map();
          Object.defineProperty(window,'localStorage',{value:{getItem:k=>data.get(k)??null,
            setItem:(k,v)=>data.set(k,String(v)),removeItem:k=>data.delete(k)}});
          window.audioEvents=[];
          const node=()=>({connect(){},disconnect(){},start(){},stop(){},
            addEventListener(){},gain:{setValueAtTime(){},linearRampToValueAtTime(){},
            exponentialRampToValueAtTime(){},cancelScheduledValues(){}},
            frequency:{setValueAtTime(){}},Q:{},type:'sine'});
          window.AudioContext=class{
            constructor(){this.state='running';this.currentTime=0;this.sampleRate=44100;this.destination={};}
            createOscillator(){const o=node();o.frequency.setValueAtTime=f=>audioEvents.push(['note',f]);return o;}
            createGain(){return node();}
            createBiquadFilter(){return {...node(),frequency:{value:0},Q:{value:0}};}
            createBuffer(channels,frames){return {getChannelData:()=>new Float32Array(frames)};}
            createBufferSource(){audioEvents.push(['tick']);return {...node(),buffer:null};}
          };
        }''')
        page.add_script_tag(content=(root/'internal/server/ui/js/i18n.js').read_text())
        page.add_script_tag(content=ns['browser_bundle']())
        page.locator('#open-settings').click()
        assert page.locator('#progress-milestones').is_checked() is False
        assert page.locator('#progress-ticking').is_checked() is False
        assert page.locator('#progress-min-duration').input_value()=='60'
        assert page.locator('#progress-volume').input_value()=='35'
        assert page.locator('#progress-sound-threshold').is_hidden()
        assert page.locator('#progress-min-duration option').count()==4
        assert page.locator('#progress-min-duration option[value="30"]').count()==0
        page.locator('#progress-milestones').check()
        assert page.locator('#progress-sound-threshold').is_visible()
        page.locator('#progress-min-duration').select_option('120')
        page.locator('#progress-ticking').check()
        page.locator('#progress-volume').fill('45')
        saved=page.evaluate("JSON.parse(localStorage.getItem('nexttimer.progress-audio.v1'))")
        assert saved=={'milestones':True,'ticking':True,'minDuration':120,'volume':45},saved
        page.locator('#preview-progress-milestones').click()
        page.wait_for_function('window.audioEvents.filter(x=>x[0]==="note").length===3')
        notes=page.evaluate('audioEvents.filter(x=>x[0]==="note").map(x=>x[1])')
        assert notes==[523.25,659.25,783.99],notes
        page.locator('#preview-progress-ticking').click()
        page.wait_for_function('audioEvents.filter(x=>x[0]==="tick").length===3')
        assert page.locator('#progress-audio-warning').is_hidden()
        page.locator('#language-select').select_option('en')
        page.wait_for_function('document.querySelector("#progress-sound-title").textContent==="Sounds"')
        assert page.locator('label[for="progress-min-duration"]').inner_text()=='Minimum duration for progress cues'
        page.locator('#language-select').select_option('de')
        page.wait_for_function('document.querySelector("#progress-sound-title").textContent==="Töne"')
        with page.expect_download() as download:
            page.locator('#standalone-export').click()
        backup=json.loads(Path(download.value.path()).read_text())
        assert 'progressAudio' not in backup and 'progress_audio' not in backup and 'volume' not in backup
        assert 'nexttimer.progress-audio.v1' not in json.dumps(backup)
        # Reopening Settings must not reset local sound preferences.
        page.locator('[data-close-dialog="settings-dialog"]').first.click()
        page.locator('#open-settings').click()
        assert page.locator('#progress-ticking').is_checked()
        assert page.locator('#progress-min-duration').input_value()=='120'
        assert not errors,errors
        assert page.evaluate('document.documentElement.scrollWidth<=innerWidth+1')
        print(f'Progress settings {width}px: local persistence, sound previews, i18n, backup exclusion, layout OK')
        page.close()
    browser.close()
