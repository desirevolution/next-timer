#!/usr/bin/env python3
"""Release-trigger and build-label regression for GitHub Pages and Local."""
from pathlib import Path
from tempfile import TemporaryDirectory
from subprocess import run
p=Path(__file__).resolve().parents[1]
pages=(p/'.github/workflows/pages.yml').read_text()
assert 'types: [published]' in pages
assert 'branches: [main]' not in pages and 'workflow_dispatch' not in pages
assert 'if: ${{ !github.event.release.prerelease }}' in pages
assert 'github.event.release.tag_name' in pages
release=(p/'.github/workflows/release.yml').read_text()
assert 'NEXTTIMER_VERSION=${{ github.event.release.tag_name }}' in release
assert 'internal/server.AppVersion=$RELEASE_TAG' in release
with TemporaryDirectory() as temp:
    output=Path(temp)/'pages'
    run(['python3',str(p/'scripts/build-static.py'),'--output',str(output),'--version','v9.8.7'],check=True)
    assert '>v9.8.7</span>' in (output/'index.html').read_text()
    sw=(output/'sw.js').read_text()
    assert "const VERSION = 'static-v9.8.7'" in sw
    assert 'recent-timers.js?v=v9.8.7' in sw
    assert (output/'js/recent-timers.js').exists()
print('Release-only Pages, stable filter, build labels & recent cache OK')
