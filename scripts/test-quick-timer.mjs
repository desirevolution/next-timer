import assert from 'node:assert/strict';
import {parseQuickTimer} from '../internal/server/ui/js/quick-timer.js';

for (const [input,duration,name] of [
  ['5m','5m',''],['3m Tee','3m','Tee'],['30s','30s',''],
  ['1h 15m','1h15m',''],['25m Pomodoro','25m','Pomodoro'],
  ['1h30m Meeting','1h30m','Meeting'],['1.5h Planung','1.5h','Planung'],
  ['  2h  10m   Training  ','2h10m','Training'],
  ['1m 30s Tee','1m30s','Tee'],['10m 2 Leute','10m','2 Leute'],
]) assert.deepEqual(parseQuickTimer(input),{duration,name},input);
for (const input of ['', 'Tee', '5mTee', '500ms Too short', '169h Too long', '0m',
  '1m '+('a'.repeat(81)), '5min Test', '0.5s test']) {
  assert.throws(()=>parseQuickTimer(input), Error, input);
}
console.log('Quick timer parser: duration+name, whitespace, 1s–168h limits, validation OK');
