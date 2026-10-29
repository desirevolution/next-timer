#!/usr/bin/env python3
"""Verify the browser-only build has its own PWA label without changing its identity."""
import json
from pathlib import Path
from tempfile import TemporaryDirectory
from urllib.parse import urljoin
import importlib.util
spec = importlib.util.spec_from_file_location('build_static', Path(__file__).with_name('build-static.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
build, SOURCE = module.build, module.SOURCE

server_manifest = json.loads((SOURCE / 'manifest.webmanifest').read_text())
assert server_manifest['name'] == 'NextTimer'
assert server_manifest['short_name'] == 'NextTimer'

with TemporaryDirectory() as directory:
    target = Path(directory) / 'static'
    build(target, 'test')
    manifest = json.loads((target / 'manifest.webmanifest').read_text())
    index = (target / 'index.html').read_text()
    assert manifest['name'] == 'NextTimer Local'
    assert manifest['short_name'] == 'NT Local'
    assert manifest['start_url'] == './?edition=local' and manifest['scope'] == './'
    # Without an explicit id, Chromium derives the app identity from start_url.
    # The edition query must distinguish server/local at the SAME URL while
    # preserving independent identities for installations under subpaths.
    assert 'id' not in manifest
    assert '<title>NextTimer Local</title>' in index
    assert 'name="apple-mobile-web-app-title" content="NextTimer Local"' in index
    assert 'href="manifest.webmanifest?v=test"' in index
    assert "'manifest.webmanifest?v=test'" in (target / 'sw.js').read_text()
    paths = ('/', '/next-timer/', '/another-copy/')
    for path in paths:
        base = 'https://timer.example.com' + path
        local_id = urljoin(base, manifest['start_url'])
        server_id = urljoin(base, server_manifest['start_url'])
        assert local_id != server_id, f'Local and server app collision: {path}'
        assert local_id == base + '?edition=local'
    assert len({urljoin('https://timer.example.com' + p, manifest['start_url']) for p in paths}) == 3
    # This must remain stable between releases; never use version in identity.
    second = Path(directory) / 'second'
    build(second, 'different-version')
    assert json.loads((second / 'manifest.webmanifest').read_text())['start_url'] == manifest['start_url']
print('Standalone PWA naming and path-based identity OK')
