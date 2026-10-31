#!/usr/bin/env python3
"""DOM regression for NextTimer dialogs, grouping, settings and one-click save."""
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
    'recent':('recent-timers.js',['RecentTimers']),
}
def bundle():
    chunks=[]
    for module,(file,exports) in MODULES.items():
        src=(UI/'js'/file).read_text().replace('import.meta.url','"https://nexttimer.example.test/js/alerts.js"')
        src=re.sub(r"^import\s+\{[^}]*\}\s+from\s+['\"][^'\"]+['\"];?\s*$",'',src,flags=re.M)
        src=re.sub(r'\bexport\s+(?=(?:async\s+)?(?:function|class|const)\b)','',src)
        pre='const {parseTimerDuration}=module_local;' if module in ['standalone','quick'] else ''
        chunks.append(f'const module_{module}=(()=>{{{pre}\n{src}\nreturn {{{",".join(exports)}}};}})();')
    app=(UI/'js/app.js').read_text()
    for filename,variable in {'favorite-sort':'sort','standalone-store':'standalone','local-timers':'local','quick-timer':'quick','alerts':'alerts','recent-timers':'recent'}.items():
        pattern=rf'^import\s+\{{([^}}]+)\}}\s+from\s+[\'\"]\./{filename}\.js\?v=[^\'\"]+[\'\"];?\s*$'
        app,count=re.subn(pattern,lambda m:f'const {{{m.group(1)}}}=module_{variable};',app,count=1,flags=re.M)
        assert count==1,filename
    return '\n'.join(chunks)+'\n'+app
html=(UI/'index.html').read_text()
html=re.sub(r'<script[^>]+src="[^"]+"[^>]*></script>','',html)
html=re.sub(r'<link[^>]+(?:rel="(?:stylesheet|manifest|icon)")[^>]*>','',html)
html=html.replace('</head>',f'<style>{(UI/"styles.css").read_text()}</style></head>')
with sync_playwright() as pw:
    browser=pw.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox'])
    for width in (1100,375):
        page=browser.new_page(viewport={'width':width,'height':820},is_mobile=width==375,has_touch=width==375)
        errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
        page.set_content(html)
        page.evaluate("""() => {const d=new Map();Object.defineProperty(window,'localStorage',{value:{getItem:k=>d.get(k)||null,setItem:(k,v)=>d.set(k,''+v),removeItem:k=>d.delete(k),clear:()=>d.clear()}})}""")
        page.add_script_tag(content=bundle())
        page.wait_for_function("document.querySelector('#quick-input') && document.querySelector('#standalone-export')")
        assert page.locator('.connection').count()==0
        assert page.locator('#others-section').evaluate('el=>!el.open')
        assert not page.locator('#new-section').count()
        assert page.locator('#backup-settings').is_visible() is False
        page.locator('#open-settings').click()
        assert page.locator('#settings-dialog').evaluate('el=>el.open')
        assert page.locator('#backup-settings').is_visible()
        page.locator('[data-close-dialog="settings-dialog"]').first.click()
        assert not page.locator('#settings-dialog').evaluate('el=>el.open')
        page.locator('#new-favorite').click()
        assert page.locator('#edit-name').input_value()==''
        assert page.locator('#edit-duration').input_value()==''
        page.locator('#edit-name').fill('Tee')
        page.locator('#edit-duration').fill('3m')
        page.locator('#edit-group').fill('Küche')
        page.locator('#edit-submit').click()
        page.wait_for_function("document.querySelectorAll('.fav-card').length===1")
        assert not page.locator('#favorite-dialog').evaluate('el=>el.open')
        assert 'Küche' in page.locator('#favorites').inner_text()
        page.locator('#sort-toggle').click()
        page.locator('[data-action="new-fav-group"]').click()
        assert page.locator('#edit-group').input_value()=='Küche'
        page.locator('#edit-name').fill('Eier')
        page.locator('#edit-duration').fill('5m')
        page.locator('#edit-submit').click()
        page.wait_for_function("document.querySelectorAll('.fav-card').length===2")
        page.locator('[data-action="rename-group"]').click()
        assert page.locator('#group-dialog').evaluate('el=>el.open')
        assert page.locator('#group-rename').input_value()=='Küche'
        field=page.locator('#group-rename').bounding_box();actions=page.locator('#group-form .dialog-footer').bounding_box()
        assert field and actions and actions['y']>=field['y']+field['height']+6,(field,actions)
        page.locator('#group-rename').fill('Kochen')
        page.locator('#group-submit').click()
        page.wait_for_function("document.querySelector('.favorite-group-heading h3')?.textContent==='Kochen'")
        page.locator('[data-action="delete-fav"]').first.click()
        assert page.locator('#confirm-dialog').evaluate('el=>el.open')
        page.locator('#confirm-cancel').click()
        assert page.locator('.fav-card').count()==2
        page.locator('#sort-toggle').click()
        page.locator('#quick-input').fill('7m Nudeln')
        page.locator('#quick-input').press('Enter')
        page.wait_for_function("document.querySelector('#hero-name').textContent==='Nudeln'")
        page.locator('#hero-favorite').click()
        assert not page.locator('#timer-actions-dialog').count()
        assert page.locator('#edit-name').input_value()=='Nudeln'
        assert page.locator('#edit-duration').input_value()=='7m'
        page.locator('#edit-submit').click()
        page.wait_for_function("document.querySelectorAll('.fav-card').length===3")
        assert page.locator('#hero-name').inner_text()=='Nudeln'
        page.locator('#quick-input').fill('2m Zweiter')
        page.locator('#quick-input').press('Enter')
        page.wait_for_function("document.querySelector('#timer-count').textContent==='1'")
        assert page.locator('#others-section').evaluate('el=>el.open')
        page.locator('[data-action="save-as-favorite"]').last.click()
        assert page.locator('#favorite-dialog').evaluate('el=>el.open')
        assert page.locator('#edit-name').input_value()=='Nudeln'
        assert page.locator('#edit-duration').input_value()=='7m'
        page.locator('#edit-cancel').click()
        page.locator('#others-section summary').click()
        assert not page.locator('#others-section').evaluate('el=>el.open')
        page.wait_for_timeout(350)
        assert not page.locator('#others-section').evaluate('el=>el.open')
        assert page.evaluate('document.documentElement.scrollWidth <= innerWidth')
        assert not errors,errors
        print(f'NextTimer UX DOM {width}px: settings, create/edit, group rename without overlap, confirmation, timer save, others auto-open OK')
        page.close()
    browser.close()
