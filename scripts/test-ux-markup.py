#!/usr/bin/env python3
"""Cheap CI guard for the shared NextTimer UX, without a browser dependency."""
from pathlib import Path
import re
root=Path(__file__).resolve().parents[1]
ui=root/'internal/server/ui'
html=(ui/'index.html').read_text()
js=(ui/'js/app.js').read_text()
for marker in ('id="quick-options"','id="new-favorite"','id="favorite-dialog"','id="settings-dialog"',
               'id="hero-favorite"','id="confirm-dialog"','id="group-dialog"','id="others-section"',
               'id="notifications-enable"','id="backup-settings"','id="edit-browser-only"'):
    assert marker in html,marker
assert 'id="new-section"' not in html
assert 'id="save-favorite"' not in html
assert "confirm(" not in js and "prompt(" not in js and "alert(" not in js
assert 'previousOtherCount' in js and "'save-as-favorite'" in js and "'new-fav-group'" in js
assert 'src="js/app.js?v=41"' in html
assert 'id="timer-actions-dialog"' not in html
assert 'data-action="timer-menu"' not in html
assert "'timer-menu'" not in js
files=[ui/'index.html',ui/'sw.js',ui/'pwa-update.js',*(ui/'js').glob('*.js')]
versions={v for f in files for v in re.findall(r'\?v=([A-Za-z0-9._-]+)',f.read_text())}
assert versions=={'41'},versions
assert "const VERSION = 'v41'" in (ui/'sw.js').read_text()
assert '?v=40' not in ''.join(f.read_text() for f in files)
print('Web UI structure and updated assets OK')
