import assert from 'node:assert/strict';
import {parseQuickTimer} from '../internal/server/ui/js/quick-timer.js';
import {durationMilliseconds, formatDurationText, parseTimerDuration} from '../internal/server/ui/js/local-timers.js';

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
for (const [raw,expected] of [
  ['3m0s','3m'], ['180000ms','3m'], ['5400000ms','1h30m'], ['1h30m0s','1h30m'],
  ['1.5s','1s500ms'], ['3601000ms','1h1s'], [1500,'1s500ms'],
]) assert.equal(formatDurationText(raw),expected,String(raw));
assert.equal(formatDurationText('nope'),'nope');
assert.equal(formatDurationText('  '),'');
assert.equal(formatDurationText(0),'');
assert.equal(formatDurationText(-1),'');
assert.equal(formatDurationText('0s'),'0s');
assert.equal(durationMilliseconds('3m0s'),180000);
assert.equal(Number.isNaN(durationMilliseconds('nope')),true);
assert.equal(parseTimerDuration('3m0s'),180000);
assert.throws(()=>parseTimerDuration('500ms'),/1s und 168h/);
console.log('Quick timer parser + duration notation: round-trips, 1s–168h limits, validation OK');
