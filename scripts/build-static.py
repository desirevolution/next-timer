#!/usr/bin/env python3
"""Build a self-contained, path-relative, browser-only NextTimer PWA.

No bundler or Go runtime required. The result can be hosted at any URL prefix.
"""
from __future__ import annotations
import argparse
import json
from pathlib import Path
import re
import shutil
import sys

ROOT=Path(__file__).resolve().parents[1]
SOURCE=ROOT/'internal/server/ui'


def build(destination:Path,version:str)->None:
    dest=destination.resolve()
    if dest==SOURCE.resolve() or SOURCE.resolve() in dest.parents:
        raise ValueError('Output directory must not be inside the source web UI')
    if not re.fullmatch(r'[a-zA-Z0-9._-]{1,80}',version):
        raise ValueError('Version must contain only letters, digits, dots, hyphens and underscores')
    if dest.exists(): shutil.rmtree(dest)
    shutil.copytree(SOURCE,dest)
    index=dest/'index.html'
    page=index.read_text()
    page=page.replace('<html lang="de">','<html lang="de" data-mode="standalone">')
    # This edition has a separate launcher label and window title. Keep the
    # shared server UI source and its PWA manifest unchanged.
    page=page.replace('<title>NextTimer</title>',
        '<title>NextTimer Local</title>\n  <meta name="apple-mobile-web-app-title" content="NextTimer Local">',1)
    manifest_path=dest/'manifest.webmanifest'
    manifest=json.loads(manifest_path.read_text())
    manifest.update({
        'name':'NextTimer Local',
        'short_name':'NT Local',
        'description':'Countdown-Timer mit lokalen Favoriten, ohne Server',
        # Without an explicit manifest id, the browser derives the identity
        # from start_url. A stable edition-specific query keeps Local distinct
        # from the server PWA without hard-coding the deployment path. This
        # also works at /, /next-timer/, or any self-hosted URL prefix.
        'start_url':'./?edition=local',
    })
    # Do not set a fixed root-relative manifest id: it would collide when
    # multiple standalone installs share an origin but use different paths.
    # Keep this query parameter unchanged in future releases: it is the
    # persisted installation identity and must not include the build version.
    manifest_path.write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
    # In standalone both quick timers and favorites are always browser-only.
    # The edit checkbox lives in the optional dialog, the quick checkbox in
    # the inline advanced options. Preserve the inputs for shared JS handlers.
    for element_id in ('timer-browser-only', 'edit-browser-only'):
        pattern = r'<label class="check-option"><input id="'+element_id+r'"[^>]*>.*?</label>'
        page, count = re.subn(pattern, f'<input id="{element_id}" type="checkbox" checked hidden>', page, count=1, flags=re.S)
        if count != 1: raise ValueError(f'Missing local-only checkbox: {element_id}')
    # Hide the now-empty advanced option instead of leaving a misleading UI.
    page = re.sub(r'<fieldset class="timer-options"><legend>Timer</legend>\s*<input id="timer-browser-only"[^>]*>\s*</fieldset>',
                  '<input id="timer-browser-only" type="checkbox" checked hidden>', page, count=1)
    page = re.sub(r'<details class="dialog-optional" id="favorite-more-options">.*?<input id="edit-browser-only"[^>]*>\s*</details>',
                  '<input id="edit-browser-only" type="checkbox" checked hidden>', page, count=1, flags=re.S)
    # A standalone install has no server connection to report. Remove the
    # entire status item (including its dot) rather than showing "Nur Browser".
    # Keep the source header untouched so the server build retains its state.
    page, count = re.subn(r'<div class="connection" aria-live="polite">.*?</div>', '', page, count=1, flags=re.S)
    if count != 1: raise ValueError('Missing server connection indicator')
    # Keep backup tools inside Settings, not permanently on the start page.
    backup='''<section class="settings-section" id="backup-settings"><h3>Daten &amp; Sicherung</h3>
      <p class="subtle">Favoriten und lokale Timer liegen nur in diesem Browser.</p>
      <div class="form-actions"><button type="button" id="standalone-export">JSON-Backup herunterladen</button>
      <label class="backup-import" for="standalone-import">JSON-Backup importieren</label>
      <input id="standalone-import" type="file" accept="application/json,.json" hidden></div></section>'''
    page, count = re.subn(r'<section class="settings-section" id="backup-settings" hidden></section>',backup,page,count=1)
    if count != 1: raise ValueError('Missing backup slot in Settings')
    page=page.replace('__NEXTTIMER_VERSION__',version)
    index.write_text(page)
    # The version is part of the cache key, not the asset names. A new
    # deployment's worker will compare versions and offer an explicit update.
    # Tag all asset URLs with the build revision so browser/CDN caches cannot
    # mix JS from an older deployment with a new HTML shell.
    for asset in [dest/'index.html',dest/'sw.js',dest/'pwa-update.js',*(dest/'js').glob('*.js')]:
        content = asset.read_text().replace('?v=40',f'?v={version}')
        # The manifest URL changes on each build to avoid stale install
        # metadata in browsers/CDNs. The app identity itself stays stable.
        if asset.name in ('index.html', 'sw.js'):
            content = content.replace('manifest.webmanifest', f'manifest.webmanifest?v={version}')
        asset.write_text(content)
    worker=dest/'sw.js'
    text=worker.read_text().replace("const VERSION = 'v40';",f"const VERSION = 'static-{version}';")
    worker.write_text(text)
    # A repair page outside the SW fetch handler, scoped to this installation.
    (dest/'pwa-repair.html').write_text('''<!doctype html><html lang="de"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>NextTimer – PWA reparieren</title><style>body{font:16px/1.6 system-ui;max-width:34rem;margin:5rem auto;padding:0 1rem;background:#0b1019;color:#eee}button{padding:.8rem;border:0;border-radius:7px;cursor:pointer}</style>
<h1>NextTimer – PWA reparieren</h1><p>Entfernt nur den Offline-Cache und den Service Worker dieser Installation. Deine lokalen Timer und Favoriten bleiben gespeichert.</p>
<button id="repair">PWA-Cache zurücksetzen</button><p id="result" role="status"></p>
<script>document.querySelector('#repair').onclick=async()=>{let b=document.querySelector('#repair');b.disabled=true;try{const root=new URL('./',location.href);const regs=await navigator.serviceWorker.getRegistrations();await Promise.all(regs.filter(r=>r.scope===root.href).map(r=>r.unregister()));const keys=await caches.keys();const prefix='nexttimer-pwa-'+encodeURIComponent(root.pathname)+'-';await Promise.all(keys.filter(k=>k.startsWith(prefix)).map(k=>caches.delete(k)));location.replace(root.href+'?repaired=1')}catch(e){b.disabled=false;document.querySelector('#result').textContent=e.message}};</script></html>''')
    # Basic source checks: don't accidentally ship server-only absolute URLs.
    for path in (dest/'index.html',dest/'manifest.webmanifest',dest/'sw.js',dest/'pwa-update.js'):
        content=path.read_text()
        if any(x in content for x in ('href="/','src="/','register(\'/sw.js\'','start_url": "/"')):
            raise ValueError(f'Root-absolute URL found in {path}')
    print(f'Standalone NextTimer {version} -> {dest}')

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output',default='dist/static')
    parser.add_argument('--version',default='dev')
    args=parser.parse_args()
    try:build(ROOT/args.output,args.version)
    except (ValueError,OSError) as e:
        print(e,file=sys.stderr);sys.exit(1)
