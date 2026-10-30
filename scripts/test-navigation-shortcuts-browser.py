#!/usr/bin/env python3
"""NextTimer interaction regression using the real standalone app modules and browser DOM."""
from pathlib import Path
from playwright.sync_api import sync_playwright
root=Path(__file__).resolve().parents[1]
# Reuse the established app module bundler, but not the executable tests below it.
quicktest=(root/'scripts/test-quick-start-browser.py').read_text()
namespace={'__file__':str(root/'scripts/test-quick-start-browser.py')}
exec(quicktest.split('with sync_playwright() as pw:')[0], namespace)
html=namespace['html']; bundle=namespace['browser_bundle']()

with sync_playwright() as pw:
    browser=pw.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox'])
    for width in (1100,375):
        page=browser.new_page(viewport={'width':width,'height':800},is_mobile=width<500,has_touch=width<500)
        errors=[]
        page.on('pageerror',lambda e:errors.append(str(e)))
        page.set_content(html)
        page.evaluate("""() => {const d=new Map();Object.defineProperty(window,'localStorage',{value:{getItem:k=>d.get(k)??null,setItem:(k,v)=>d.set(k,''+v),removeItem:k=>d.delete(k)}});}""")
        page.add_script_tag(content=bundle)
        page.wait_for_function("document.querySelector('#quick-input') && document.querySelector('#standalone-export')")
        page.locator('#new-favorite').click()
        page.locator('#edit-name').fill('Tee')
        # Arrows in editing dialogs must move the caret, not the button focus.
        page.locator('#edit-name').press('ArrowLeft')
        assert page.locator('#edit-name').evaluate('el=>el.selectionStart')==2
        assert page.locator('#edit-name').evaluate('el=>el===document.activeElement')
        page.locator('#edit-duration').fill('4m')
        page.locator('#edit-shortcut').select_option('1')
        page.locator('#edit-submit').click()
        page.wait_for_function("document.querySelector('.favorite-shortcut')?.textContent==='#1'")
        page.locator('#new-favorite').click()
        page.locator('#edit-name').fill('Anderer')
        page.locator('#edit-duration').fill('2m')
        page.locator('#edit-shortcut').select_option('1')
        page.locator('#edit-submit').click()
        assert 'bereits vergeben' in page.locator('#edit-error').inner_text()
        page.locator('#edit-shortcut').select_option('0')
        page.locator('#edit-submit').click()
        page.wait_for_function("document.querySelectorAll('.favorite-shortcut').length===2")
        assert sorted(page.locator('.favorite-shortcut').all_text_contents())==['#0','#1']
        assert page.locator('.connection').count()==0
        field=page.locator('#quick-input')
        field.fill('#1');field.press('Enter')
        page.wait_for_function("document.querySelector('#hero-name').textContent==='Tee'")
        field.fill('#9');field.press('Enter')
        assert 'Kein Favorit' in page.locator('#quick-error').inner_text()
        field.press('Escape')
        assert field.input_value()=='' and field.evaluate('el=>el===document.activeElement')
        page.wait_for_timeout(25)
        field.fill('8m Acht');field.press('Enter')
        page.wait_for_function("document.querySelector('#timer-count').textContent==='1'")
        page.wait_for_timeout(25)
        field.fill('2m Zwei');field.press('Enter')
        page.wait_for_function("document.querySelector('#timer-count').textContent==='2'")
        assert page.locator('#hero-name').inner_text()=='Zwei'  # automatic: soonest first
        assert page.locator('#focus-position').inner_text()=='3 von 3'
        # Escape is global on the main screen, even with focus on a favorite,
        # other button or one of the advanced new-timer controls.
        page.locator('.fav-start').first.focus()
        page.keyboard.press('Escape')
        assert page.locator('#focus-panel').evaluate('el=>el===document.activeElement')
        page.keyboard.press('Escape')
        assert field.evaluate('el=>el===document.activeElement')
        page.locator('[data-quick-duration="5m"]').focus()
        page.keyboard.press('Escape')
        assert page.locator('#focus-panel').evaluate('el=>el===document.activeElement')
        page.keyboard.press('Escape')
        page.locator('#quick-options').evaluate('el=>el.open=true')
        page.locator('#timer-browser-only').focus()
        page.keyboard.press('Escape')
        assert page.locator('#focus-panel').evaluate('el=>el===document.activeElement')
        page.locator('#hero-toggle').focus()
        page.keyboard.press('Escape')
        assert field.evaluate('el=>el===document.activeElement')
        page.locator('#quick-options').evaluate('el=>el.open=false')
        # Clear text first, then navigate from an empty entry.
        field.fill('10m irgendwas');field.press('Escape')
        assert field.input_value()==''
        field.press('Escape')
        assert page.locator('#focus-panel').evaluate('el=>el===document.activeElement')
        page.keyboard.press('ArrowRight')
        assert page.locator('#hero-name').inner_text()=='Tee'
        page.keyboard.press('ArrowRight')
        assert page.locator('#hero-name').inner_text()=='Acht'
        page.keyboard.press('ArrowRight')
        assert page.locator('#hero-name').inner_text()=='Zwei'  # wrap
        page.keyboard.press('p')
        page.wait_for_function("document.querySelector('#head-status').textContent==='Pausiert'")
        assert page.locator('#hero-name').inner_text()=='Zwei'  # pinned despite paused
        page.keyboard.press('p')
        page.wait_for_function("document.querySelector('#head-status').textContent==='Läuft'")
        page.keyboard.press('s')
        assert page.locator('#confirm-dialog').evaluate('el=>el.open')
        assert 'Zwei' in page.locator('#confirm-description').inner_text()
        # Left always focuses cancel; right always focuses the destructive
        # action. Enter activates the highlighted button; Esc only cancels.
        assert page.locator('#confirm-cancel').evaluate('el=>el===document.activeElement')
        page.keyboard.press('ArrowRight')
        assert page.locator('#confirm-ok').evaluate('el=>el===document.activeElement')
        page.keyboard.press('ArrowLeft')
        assert page.locator('#confirm-cancel').evaluate('el=>el===document.activeElement')
        page.keyboard.press('Enter')
        assert not page.locator('#confirm-dialog').evaluate('el=>el.open')
        assert page.locator('#hero-name').inner_text()=='Zwei'
        # All confirmations use the same dialog; Escape never stops a timer.
        page.locator('#hero-stop').click()
        page.keyboard.press('ArrowRight')
        page.keyboard.press('Escape')
        assert not page.locator('#confirm-dialog').evaluate('el=>el.open')
        assert page.locator('#hero-name').inner_text()=='Zwei'
        page.locator('#hero-stop').click()
        assert page.locator('#confirm-dialog').evaluate('el=>el.open')
        page.keyboard.press('ArrowRight')
        assert page.locator('#confirm-ok').evaluate('el=>el===document.activeElement')
        page.keyboard.press('Enter')
        page.wait_for_function("document.querySelector('#hero-name').textContent==='Tee'") # automatic after deleted
        assert page.locator('#focus-position').inner_text()=='1 von 2'
        page.locator('#focus-previous').click()
        assert page.locator('#hero-name').inner_text()=='Acht'
        page.locator('#focus-next').click()
        assert page.locator('#hero-name').inner_text()=='Tee'
        page.locator('#focus-panel').focus()
        page.keyboard.press('Escape')
        assert field.evaluate('el=>el===document.activeElement')
        page.locator('#focus-panel').focus()
        page.keyboard.press('Shift+/')
        assert page.locator('#shortcuts-dialog').evaluate('el=>el.open')
        page.keyboard.press('Escape')
        assert not page.locator('#shortcuts-dialog').evaluate('el=>el.open')
        page.locator('#open-shortcuts').click()
        assert page.locator('#shortcuts-dialog').evaluate('el=>el.open')
        page.keyboard.press('Escape')
        # The shared confirmation dialog is also used for deleting favorites.
        # Arrow navigation is deliberately not attached to editor dialogs.
        page.locator('#sort-toggle').click()
        page.locator('[data-action="delete-fav"]').first.click()
        assert page.locator('#confirm-dialog').evaluate('el=>el.open')
        assert page.locator('#confirm-cancel').evaluate('el=>el===document.activeElement')
        page.keyboard.press('ArrowRight')
        assert page.locator('#confirm-ok').evaluate('el=>el===document.activeElement')
        page.keyboard.press('Escape')
        assert not page.locator('#confirm-dialog').evaluate('el=>el.open')
        assert page.locator('.fav-card').count()==2
        page.locator('#sort-toggle').click()
        # Both timers are paused: the fallback must show a paused timer rather
        # than the empty state. Also test that stopping the last one clears it.
        page.locator('#hero-toggle').click() # Tee
        page.wait_for_function("document.querySelector('#head-status').textContent==='Pausiert'")
        page.locator('#focus-next').click() # Acht
        page.locator('#hero-toggle').click()
        page.wait_for_function("document.querySelector('#head-status').textContent==='Pausiert'")
        page.locator('#hero-stop').click();page.locator('#confirm-ok').click()
        page.wait_for_function("document.querySelector('#hero-name').textContent==='Tee'")
        assert page.locator('#head-status').inner_text()=='Pausiert'
        page.locator('#hero-stop').click();page.locator('#confirm-ok').click()
        page.wait_for_function("document.querySelector('#hero-name').textContent==='Noch kein Timer aktiv'")
        if width==375:
            # Simulate Android's shrunken visible viewport when keyboard opens.
            for opener,footer in [('#new-favorite','#favorite-dialog .dialog-footer'),('#open-settings','#settings-dialog .dialog-footer')]:
                page.locator(opener).click()
                page.evaluate("document.documentElement.style.setProperty('--dialog-viewport-height','390px')")
                bounds=page.locator(footer).bounding_box()
                assert bounds and bounds['y']+bounds['height'] <=391,(opener,bounds)
                assert page.locator(footer).is_visible()
                page.locator(f"{footer.split(' ')[0]} [data-close-dialog]").first.click() if opener=='#open-settings' else page.locator('#edit-cancel').click()
        assert page.evaluate('document.documentElement.scrollWidth <= innerWidth')
        assert not errors,errors
        print(f'Keyboard/favorites/timer switch/stop/mobile dialogs: {width}px OK')
        page.close()
    browser.close()
