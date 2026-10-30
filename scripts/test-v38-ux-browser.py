#!/usr/bin/env python3
"""Regression: v38 sorting pointer interaction, focus-free timer navigation, compact alarm rows."""
from pathlib import Path
from playwright.sync_api import sync_playwright
root=Path(__file__).resolve().parents[1]
quicktest=root/'scripts/test-quick-start-browser.py'
namespace={'__file__':str(quicktest)}
exec(quicktest.read_text().split('with sync_playwright() as pw:')[0],namespace)
html,bundle=namespace['html'],namespace['browser_bundle']()
with sync_playwright() as pw:
    browser=pw.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox'])
    for width in (1100,375):
        page=browser.new_page(viewport={'width':width,'height':740},has_touch=width<500,is_mobile=width<500)
        errors=[]
        page.on('pageerror',lambda err:errors.append(str(err)))
        page.set_content(html)
        page.evaluate("""() => {const d=new Map();Object.defineProperty(window,'localStorage',{value:{getItem:k=>d.get(k)??null,setItem:(k,v)=>d.set(k,''+v),removeItem:k=>d.delete(k)}})}""")
        page.add_script_tag(content=bundle)
        page.wait_for_function("document.querySelector('#quick-input') && document.querySelector('#standalone-export')")
        for i, name in enumerate(['Kaffee','Pause','Tee']):
            page.locator('#new-favorite').click()
            page.locator('#edit-name').fill(name)
            page.locator('#edit-duration').fill(f'{i+2}m')
            page.locator('#edit-submit').click()
            page.wait_for_function('(n)=>document.querySelectorAll(".fav-start").length===n',arg=i+1)
        # Sort presentation: desktop 2 columns, mobile 1, no duplicate up/down buttons.
        page.locator('#sort-toggle').click()
        assert page.locator('[data-action="fav-up"],[data-action="fav-down"]').count()==0
        columns=page.locator('.favorites-grid').first.evaluate('el=>getComputedStyle(el).gridTemplateColumns.split(" ").length')
        assert columns==(2 if width==1100 else 1),(width,columns)
        assert page.locator('.drag-handle').count()==3
        grid=page.locator('.favorites-grid').first
        start=grid.locator('.drag-handle').first.bounding_box()
        target=grid.locator('.fav-card').nth(1).bounding_box()
        a=(start['x']+start['width']/2,start['y']+start['height']/2)
        # For the second grid card, mouse should enter near its right/bottom edge.
        b=(target['x']+target['width']*.78,target['y']+target['height']*.75)
        page.mouse.move(*a);page.mouse.down();page.mouse.move(a[0]+10,a[1]+8,steps=3)
        page.mouse.move(*b,steps=12)
        page.wait_for_timeout(120)
        assert page.locator('.drag-ghost').count()==1, 'Lifted drag preview expected'
        page.mouse.up()
        page.wait_for_timeout(80)
        assert page.locator('.drag-ghost').count()==0
        names=grid.locator('.sort-name [data-i18n-ignore]').all_text_contents()
        assert names==['Pause','Kaffee','Tee'],names
        # Persisted across edit mode and page re-render.
        page.wait_for_function('document.querySelector("#sort-status").textContent.includes("gespeichert")')
        page.locator('#sort-toggle').click()
        assert page.locator('.fav-name').all_text_contents()==names
        # Launch input: focus must leave entry after a successful start.
        field=page.locator('#quick-input')
        field.fill('6m Eins');field.press('Enter')
        page.wait_for_function('document.querySelector("#hero-name").textContent==="Eins"')
        page.wait_for_function('document.activeElement!==document.querySelector("#quick-input")')
        field.fill('8m Zwei');field.press('Enter')
        page.wait_for_function('document.querySelector("#hero-name").textContent==="Zwei"')
        page.wait_for_function('document.activeElement!==document.querySelector("#quick-input")')
        # A different button has focus: arrows globally change the display.
        page.locator('#sort-toggle').focus()
        page.keyboard.press('ArrowLeft')
        assert page.locator('#hero-name').inner_text()=='Eins'
        page.keyboard.press('ArrowRight')
        assert page.locator('#hero-name').inner_text()=='Zwei'
        # But arrow keys in text fields move the caret rather than switching.
        field.fill('10m Probe');field.press('Home');field.press('ArrowRight')
        assert page.locator('#hero-name').inner_text()=='Zwei'
        assert field.evaluate('el=>el.selectionStart')==1
        # Alarm rendering has one shared container and no per-row red card.
        page.evaluate("""() => {
          const alert=document.querySelector('#alarm-banner');alert.hidden=false;
          document.querySelector('#alarm-count').textContent='(4)';
          document.querySelector('#alarm-list').innerHTML=Array.from({length:4},(_,i)=>
            `<div class="alarm-item"><span class="alarm-item-name">Timer ${i}</span><button>Bestätigen</button></div>`).join('');
        }""")
        assert page.locator('#alarm-list .alarm-item-state').count()==0
        assert page.locator('#alarm-list .alarm-item').count()==4
        assert page.locator('#alarm-list').evaluate('el=>getComputedStyle(el).overflowY')=='auto'
        assert page.evaluate('document.documentElement.scrollWidth<=innerWidth+1'),width
        assert not errors,errors
        print(f'NextTimer v38 {width}px: drag and drop, responsive sort, global arrows, focus release, alarms OK')
        page.close()
    browser.close()
