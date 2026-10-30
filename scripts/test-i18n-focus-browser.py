#!/usr/bin/env python3
"""Verify automatic German/English UI, persistence, user data and newest-timer focus."""
from pathlib import Path
from playwright.sync_api import sync_playwright
root=Path(__file__).resolve().parents[1]
quicktest=(root/'scripts/test-quick-start-browser.py').read_text()
namespace={'__file__':str(root/'scripts/test-quick-start-browser.py')}
exec(quicktest.split('with sync_playwright() as pw:')[0],namespace)
html=namespace['html'];bundle=namespace['browser_bundle']()
i18n=(root/'dist/static/js/i18n.js').read_text()
with sync_playwright() as pw:
    browser=pw.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox'])
    for width in (1100,375):
        context=browser.new_context(viewport={'width':width,'height':850},locale='en-US',is_mobile=width<500,has_touch=width<500)
        page=context.new_page()
        errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
        page.set_content(html)
        page.evaluate("""() => {window._localStorage = new Map();Object.defineProperty(window,'localStorage',{value:{getItem:k=>_localStorage.get(k)??null,setItem:(k,v)=>_localStorage.set(k,''+v),removeItem:k=>_localStorage.delete(k)}})}""")
        page.add_script_tag(content=i18n)
        page.add_script_tag(content=bundle)
        page.wait_for_function("document.documentElement.lang==='en' && document.querySelector('#quick-title').textContent==='New timer'")
        assert page.locator('#quick-input').get_attribute('placeholder')=='e.g. 5m Tea or #1'
        assert not page.locator('#quick-hint').count() and not page.locator('#focus-keyboard-hint').count()
        page.locator('#open-settings').click()
        assert page.locator('#settings-title').inner_text()=='Settings'
        assert page.locator('#language-select').input_value()=='auto'
        assert page.locator('#language-select').evaluate("el=>el.closest('.settings-section')?.querySelector('h3')?.textContent")=='Language'
        page.locator('#language-select').select_option('de')
        page.wait_for_function("document.documentElement.lang==='de' && document.querySelector('#settings-title').textContent==='Einstellungen'")
        assert page.evaluate("localStorage.getItem('nexttimer.language.v1')")=='de'
        page.locator('[data-close-dialog="settings-dialog"]').first.click()
        page.locator('#quick-input').fill('3m Pausiert')
        page.locator('#quick-input').press('Enter')
        page.wait_for_function("document.querySelector('#hero-name').textContent==='Pausiert'")
        # The timer's own name must never change after switching languages.
        page.locator('#open-settings').click()
        page.locator('#language-select').select_option('en')
        page.wait_for_function("document.querySelector('#quick-title').textContent==='New timer'")
        assert page.locator('#hero-name').inner_text()=='Pausiert'
        assert page.locator('#settings-title').inner_text()=='Settings'
        page.locator('[data-close-dialog="settings-dialog"]').first.click()
        page.locator('#hero-stop').click()
        page.wait_for_function("document.querySelector('#confirm-title').textContent==='Stop timer?'")
        assert 'Pausiert' in page.locator('#confirm-description').inner_text()
        assert 'remaining' in page.locator('#confirm-description').inner_text()
        page.locator('#confirm-cancel').click()
        # New favorite start must replace a previously selected paused timer.
        page.locator('#hero-toggle').click()
        page.wait_for_function("document.querySelector('#head-status').textContent==='Paused'")
        page.locator('#new-favorite').click()
        page.locator('#edit-name').fill('Pause')
        page.locator('#edit-duration').fill('5m')
        page.locator('#edit-shortcut').select_option('1')
        dialog=page.locator('#favorite-dialog .dialog-main')
        input=page.locator('#edit-name')
        outer=dialog.bounding_box();field=input.bounding_box()
        assert outer and field and field['x']>outer['x']+3,(outer,field)
        page.locator('#edit-submit').click()
        page.wait_for_function("document.querySelector('.fav-name')?.textContent==='Pause'")
        page.locator('#quick-input').fill('#1')
        page.locator('#quick-input').press('Enter')
        page.wait_for_function("document.querySelector('#hero-name').textContent==='Pause'")
        assert page.locator('#hero-name').inner_text()=='Pause'
        assert page.locator('#head-status').inner_text()=='Running'
        # Back to the browser language after switching locally.
        page.locator('#open-settings').click()
        page.locator('#language-select').select_option('auto')
        assert page.evaluate("localStorage.getItem('nexttimer.language.v1')")=='auto'
        assert page.locator('#settings-title').inner_text()=='Settings'
        assert not errors,errors
        print(f'I18n and focus {width}px: auto/en/de, confirmations, user text, favorites, dialog insets OK')
        context.close()
    browser.close()
