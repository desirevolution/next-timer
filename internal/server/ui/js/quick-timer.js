import {parseTimerDuration} from './local-timers.js?v=35';

// Accept a sequence of duration units followed by an optional timer name.
// The normalized duration is valid for both Go's API and the local timer engine.
const unit = /(\d+(?:\.\d+)?)(ms|h|m|s)/y;

export function parseQuickTimer(value) {
  const input = String(value ?? '').trim();
  if (!input) throw Error('Dauer angeben, z. B. 5m Tee.');
  let index = 0;
  let duration = '';
  while (index < input.length) {
    const beforeWhitespace = index;
    while (index < input.length && /\s/.test(input[index])) index++;
    unit.lastIndex = index;
    const match = unit.exec(input);
    if (!match) {
      index = beforeWhitespace;
      break;
    }
    duration += match[0];
    index = unit.lastIndex;
    if (index < input.length && !/[\s\d]/.test(input[index])) {
      throw Error('Dauer und Name mit einem Leerzeichen trennen, z. B. 5m Tee.');
    }
  }
  // Use the existing duration rules, including the 1s–168h limit.
  parseTimerDuration(duration);
  const name = input.slice(index).trim();
  if ([...name].length > 80) throw Error('Name zu lang (maximal 80 Zeichen).');
  return {duration, name};
}
