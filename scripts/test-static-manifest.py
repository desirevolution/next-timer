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
    assert manifest['start_url'] == './' and manifest['scope'] == './'
    # A fixed ID would make multiple self-hosted copies under different
    # prefixes on the same origin look like the same installed app.
    assert 'id' not in manifest
    assert '<title>NextTimer Local</title>' in index
    assert 'name="apple-mobile-web-app-title" content="NextTimer Local"' in index
    assert urljoin('https://timer.example.com/', manifest['start_url']) != urljoin(
        'https://timer.example.com/local/', manifest['start_url'])
    assert urljoin('https://timer.example.com/local/', manifest['start_url']) != urljoin(
        'https://timer.example.com/second/', manifest['start_url'])
print('Standalone PWA naming and path-based identity OK')
