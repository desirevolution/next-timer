// Lightweight pointer-based reordering for mouse, pen and touch.
// Only the explicit drag handle initiates movement, so favorite tiles remain
// ordinary click targets outside sorting mode. No external JS dependency.
export function enableFavoriteSorting(container, onCommit, onCancel) {
  let active = null;
  const ids = () => [...container.querySelectorAll('.fav-card')].map(card => card.dataset.id);
  container.addEventListener('pointerdown', e => {
    const handle = e.target.closest('.drag-handle');
    const card = handle?.closest('.fav-card');
    if (!card || e.button !== 0 || active) return;
    e.preventDefault();
    active = {handle, card, pointerId:e.pointerId, x:e.clientX, y:e.clientY, changed:false};
    handle.setPointerCapture(e.pointerId);
  });
  document.addEventListener('pointermove', e => {
    if (!active || e.pointerId !== active.pointerId) return;
    const {card} = active;
    if (!active.changed && Math.hypot(e.clientX-active.x, e.clientY-active.y) < 6) return;
    active.changed = true;
    card.classList.add('dragging');
    const target = document.elementFromPoint(e.clientX,e.clientY)?.closest('.fav-card');
    if (!target || target === card || target.closest('.favorite-group') !== card.closest('.favorite-group')) return;
    const box = target.getBoundingClientRect();
    const before = e.clientX < box.left+box.width/2;
    if (before) target.before(card);
    else target.after(card);
    // Re-capture after DOM movement; moving a captured element may release it.
    try {active.handle.setPointerCapture(e.pointerId);} catch {}
  });
  function end(e, cancelled=false) {
    if (!active || e.pointerId !== active.pointerId) return;
    const previous = active;
    active = null;
    previous.card.classList.remove('dragging');
    if (previous.handle.hasPointerCapture(e.pointerId)) previous.handle.releasePointerCapture(e.pointerId);
    if (cancelled) { if(previous.changed) onCancel(); return; }
    if (previous.changed) onCommit(ids());
  }
  document.addEventListener('pointerup', e => end(e));
  document.addEventListener('pointercancel', e => end(e,true));
  return { isDragging: () => !!active };
}
