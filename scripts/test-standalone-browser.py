from playwright.sync_api import sync_playwright
from urllib.request import urlopen
from urllib.error import HTTPError
import json, time
URL='https://nexttimer.example.test/next-timer/'
with sync_playwright() as pw:
    browser=pw.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox'])
    context=browser.new_context(viewport={'width':1100,'height':780},service_workers='block')
    def from_local(route):
        try:
            url=route.request.url.replace('https://nexttimer.example.test/','http://127.0.0.1:8765/')
            with urlopen(url,timeout=5) as response:
                route.fulfill(status=response.status,headers={'Content-Type':response.headers.get('Content-Type','application/octet-stream')},body=response.read())
        except HTTPError as exc:
            route.fulfill(status=exc.code,body=str(exc).encode())
    context.route('**/*',from_local)
    page=context.new_page()
    errors=[]; api_requests=[]
    page.on('pageerror',lambda e:errors.append(str(e)))
    page.on('request',lambda r: api_requests.append(r.url) if '/api/' in r.url else None)
    page.goto(URL,wait_until='load',timeout=20000)
    page.wait_for_timeout(1500)
    print('TITLE',page.title())
    print('STATUS',page.locator('#connect').inner_text())
    print('SERVICEWORKER disabled for DOM test')
    assert page.locator('#timer-browser-only').is_hidden()
    assert page.locator('#connect').inner_text()=='Nur Browser'
    page.locator('#new-section summary').click()
    page.locator('#name').fill('Kaffee')
    page.locator('#duration').fill('15s')
    page.locator('#timer-group').fill('Küche')
    page.locator('#save-favorite').click()
    page.wait_for_timeout(350)
    print('FAVORITES',page.locator('.fav-card').count(),page.locator('.favorite-group-heading').all_text_contents())
    assert page.locator('.fav-card').count()==1
    assert 'Küche' in page.locator('.favorite-group-heading').inner_text()
    page.reload();page.wait_for_timeout(400)
    assert page.locator('.fav-card').count()==1
    page.locator('[data-action="start-fav"]').click();page.wait_for_timeout(400)
    print('TIMER',page.locator('#hero-name').inner_text(),page.locator('#hero-time').inner_text())
    assert page.locator('#hero-name').inner_text()=='Kaffee'
    assert page.locator('#hero-origin').is_hidden()
    page.locator('#sort-toggle').click()
    page.locator('[data-action="edit-fav"]').click()
    page.locator('#edit-name').fill('Kaffee lang')
    page.locator('#edit-submit').click();page.wait_for_timeout(300)
    assert 'Kaffee lang' in page.locator('#favorites').inner_text()
    page.locator('#standalone-export').click()
    page.wait_for_timeout(100)
    data=page.evaluate("JSON.parse(localStorage.getItem('nexttimer.standalone.templates.v1'))")
    assert data[0]['name']=='Kaffee lang' and data[0]['browser_only']==True
    assert not api_requests, api_requests
    print('ERRORS',errors)
    assert not errors, errors
    browser.close()
