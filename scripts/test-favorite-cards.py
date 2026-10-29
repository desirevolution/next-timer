#!/usr/bin/env python3
"""Verify the favorite-card layout without requiring a network server."""
from pathlib import Path
from playwright.sync_api import sync_playwright

ROOT = Path(__file__).resolve().parents[1]
CSS = (ROOT / 'internal/server/ui/styles.css').read_text()
APP = (ROOT / 'internal/server/ui/js/app.js').read_text()
assert '<span class="fav-play">▶ Starten</span>' not in APP
assert 'class="fav-start" type="button" data-action="start-fav"' in APP
assert 'class="sort-actions"' in APP

with sync_playwright() as pw:
    browser = pw.chromium.launch(headless=True, executable_path='/usr/bin/chromium', args=['--no-sandbox'])
    for viewport in ({'width':1100,'height':780}, {'width':375,'height':812}):
        page=browser.new_page(viewport=viewport)
        cards=''.join(f'<article class="fav-card"><button class="fav-start" type="button" data-action="start-fav" aria-label="Timer {i} starten"><span class="fav-name">{name}</span><span class="fav-duration">03:00</span></button></article>' for i,name in enumerate(('Tee','Kaffee','Pomodoro','Mittagspause','Meeting','Eier'),1))
        # The real app fills this same grid using its favoriteCard() renderer.
        page.set_content(f'<html><head><style>{CSS}</style></head><body><main class="shell"><section class="panel fav-panel"><div class="favorites-grid">{cards}</div></section></main></body></html>')
        events=[]
        page.expose_function('recordStart',lambda name:events.append(name))
        page.evaluate("document.querySelectorAll('.fav-start').forEach(el=>el.addEventListener('click',()=>window.recordStart(el.querySelector('.fav-name').textContent)))")
        button=page.locator('.fav-start').first
        bounds=button.bounding_box()
        assert bounds and bounds['height'] <= 120, bounds
        button.click(position={'x':bounds['width']-7,'y':bounds['height']/2})
        assert events==['Tee'],events
        page.locator('.fav-start').nth(1).focus()
        page.keyboard.press('Enter')
        assert events==['Tee','Kaffee'],events
        assert page.evaluate('document.documentElement.scrollWidth <= innerWidth'),'Horizontal overflow'
        columns=page.locator('.favorites-grid').evaluate('(el)=>getComputedStyle(el).gridTemplateColumns.split(" ").length')
        assert (columns>=4 if viewport['width']>800 else columns==2),columns
        if viewport['width']>800:
            assert button.evaluate("el=>getComputedStyle(el).gridTemplateColumns.split(' ').length")==2
        print(f"Favorite cards {viewport['width']}px: {columns} columns; right-edge click, keyboard start and layout OK")
        page.close()
    browser.close()
