#!/usr/bin/env python3
"""Integration of history popup, keyboard navigation and favorite blur in Local."""
from pathlib import Path
from playwright.sync_api import sync_playwright
root=Path(__file__).resolve().parents[1]
quick=root/'scripts/test-quick-start-browser.py'
ns={'__file__':str(quick)}
exec(quick.read_text().split('with sync_playwright() as pw:')[0],ns)
with sync_playwright() as pw:
    browser=pw.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox'])
    for width in (1100,375):
        page=browser.new_page(viewport={'width':width,'height':780},has_touch=width==375,is_mobile=width==375)
        errors=[];page.on('pageerror',lambda err: errors.append(str(err)))
        page.set_content(ns['html'])
        page.evaluate("""()=>{const d=new Map();Object.defineProperty(window,'localStorage',{value:{getItem:k=>d.get(k)||null,setItem:(k,v)=>d.set(k,String(v)),removeItem:k=>d.delete(k)}})}""")
        page.add_script_tag(content=ns['browser_bundle']())
        page.wait_for_function("document.querySelector('#recent-toggle')")
        assert page.locator('#app-version').inner_text()=='dev-v40'
        assert page.locator('.confirm-keyboard-hint').count()==0
        assert page.locator('#recent-toggle').is_disabled()
        field=page.locator('#quick-input')
        field.fill('5m Tee');field.press('Enter')
        page.wait_for_function('!document.querySelector("#recent-toggle").disabled')
        field.fill('2m Kaffee');field.press('Enter')
        page.wait_for_function('document.querySelector("#hero-name").textContent==="Kaffee"')
        page.locator('#recent-toggle').click()
        assert page.locator('#recent-entry').count()==0
        assert page.locator('.recent-entry').count()==2
        assert page.locator('.recent-entry-name').all_text_contents()==['Kaffee','Tee']
        page.locator('.recent-entry').nth(1).click()
        page.wait_for_function('document.querySelector("#hero-name").textContent==="Tee"')
        assert page.locator('.recent-popover').is_hidden()
        assert page.evaluate('document.activeElement===document.body')
        field.focus();field.press('ArrowUp')
        assert field.input_value()=='5m Tee'
        field.press('ArrowUp')
        assert field.input_value()=='2m Kaffee'
        field.press('ArrowDown')
        assert field.input_value()=='5m Tee'
        field.press('Enter')
        page.wait_for_function('document.querySelector("#hero-name").textContent==="Tee"')
        page.locator('#new-favorite').click()
        page.locator('#edit-name').fill('Pause')
        page.locator('#edit-duration').fill('3m')
        page.locator('#edit-submit').click()
        page.wait_for_function('document.querySelectorAll(".fav-start").length===1')
        fav=page.locator('.fav-start');fav.click()
        page.wait_for_function('document.querySelector("#hero-name").textContent==="Pause"')
        page.wait_for_function('document.activeElement!==document.querySelector(".fav-start")')
        page.locator('#recent-toggle').click()
        assert page.locator('.recent-entry-name').first.inner_text()=='Pause'
        page.locator('#recent-clear').click()
        assert page.locator('#recent-toggle').is_disabled()
        assert not errors,errors
        print(f'History + focus, viewport {width}px: OK')
        page.close()
    browser.close()
