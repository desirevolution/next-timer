#!/usr/bin/env python3
"""Cheap CI guard for the shared NextTimer UX, without a browser dependency."""
from pathlib import Path
root=Path(__file__).resolve().parents[1]
ui=root/'internal/server/ui'
html=(ui/'index.html').read_text()
js=(ui/'js/app.js').read_text()
for marker in ('id="quick-options"','id="new-favorite"','id="favorite-dialog"','id="settings-dialog"',
               'id="timer-actions-dialog"','id="confirm-dialog"','id="group-dialog"','id="others-section"',
               'id="notifications-enable"','id="backup-settings"','id="edit-browser-only"'):
    assert marker in html,marker
assert 'id="new-section"' not in html
assert 'id="save-favorite"' not in html
assert "confirm(" not in js and "prompt(" not in js and "alert(" not in js
assert 'previousOtherCount' in js and "'timer-menu'" in js and "'new-fav-group'" in js
assert 'src="js/app.js?v=30"' in html
print('Web UI structure and updated assets OK')
