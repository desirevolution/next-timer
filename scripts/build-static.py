#!/usr/bin/env python3
"""Build a self-contained, path-relative, browser-only NextTimer PWA.

No bundler or Go runtime required. The result can be hosted at any URL prefix.
"""
from __future__ import annotations
import argparse
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
    # Keep the checkbox element for the shared form handlers, but remove the
    # user-facing local/server switch. The standalone mode is not optional.
    for element_id in ('timer-browser-only','edit-browser-only'):
        pattern=r'<label class="check-option"><input id="'+element_id+r'"[^>]*>.*?</label>'
        page,count=re.subn(pattern,f'<input id="{element_id}" type="checkbox" checked hidden>',page,count=1,flags=re.S)
        if count!=1:raise ValueError(f'Cannot locate {element_id} option')
    # The server/local checkbox is not meaningful in static mode. Remove the
    # otherwise empty fieldsets but preserve their hidden inputs for form code.
    for element_id in ('timer-browser-only','edit-browser-only'):
        pattern = (r'<fieldset class="timer-options">\s*<legend>.*?</legend>\s*'
                   r'(<input id="'+element_id+r'"[^>]*>)\s*</fieldset>')
        page, count = re.subn(pattern, r'\1', page, count=1, flags=re.S)
        if count != 1: raise ValueError(f'Cannot remove standalone-only empty options fieldset: {element_id}')
    page=page.replace('Verbinde …','Nur Browser')
    # The local backup is required because clearing browser data erases all
    # favorites and timers. It is deliberately standalone-only.
    backup='''  <details class="panel extra-panel standalone-backup" id="standalone-backup">
    <summary><span>Datensicherung</span><span class="chevron" aria-hidden="true">⌄</span></summary>
    <p class="subtle">Favoriten und lokale Timer werden nur auf diesem Gerät gespeichert.</p>
    <div class="form-actions">
      <button type="button" id="standalone-export">JSON-Backup herunterladen</button>
      <label class="secondary-button backup-import" for="standalone-import">JSON-Backup importieren</label>
      <input id="standalone-import" type="file" accept="application/json,.json" hidden>
    </div>
  </details>
'''
    page=page.replace('  <p id="message"',backup+'  <p id="message"')
    index.write_text(page)
    # The version is part of the cache key, not the asset names. A new
    # deployment's worker will compare versions and offer an explicit update.
    # Tag all asset URLs with the build revision so browser/CDN caches cannot
    # mix JS from an older deployment with a new HTML shell.
    for asset in [dest/'index.html',dest/'sw.js',dest/'pwa-update.js',*(dest/'js').glob('*.js')]:
        asset.write_text(asset.read_text().replace('?v=25',f'?v={version}'))
    worker=dest/'sw.js'
    text=worker.read_text().replace("const VERSION = 'v26';",f"const VERSION = 'static-{version}';")
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
