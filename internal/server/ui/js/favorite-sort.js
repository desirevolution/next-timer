// Native pointer events, no dependency. The drag handle alone starts sorting;
// ordinary cards remain clickable. Works with mouse, pen and touch.
export function enableFavoriteSorting(container,onCommit,onCancel) {
  let active=null;
  const allIDs=()=>[...container.querySelectorAll('.fav-card:not(.drag-ghost)')].map(card=>card.dataset.id);
  const reduced=()=>window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
  function dragGhost(a) {
    const rect=a.card.getBoundingClientRect();
    const ghost=a.card.cloneNode(true);
    ghost.removeAttribute('data-id');
    ghost.setAttribute('aria-hidden','true');
    ghost.classList.add('drag-ghost');
    Object.assign(ghost.style,{width:`${rect.width}px`,height:`${rect.height}px`,left:`${rect.left}px`,top:`${rect.top}px`});
    // Prevent duplicate ids or tab stops inside this purely visual clone.
    for(const button of ghost.querySelectorAll('button'))button.tabIndex=-1;
    document.body.append(ghost);
    a.ghost=ghost;a.dx=a.x-rect.left;a.dy=a.y-rect.top;
    a.card.classList.add('dragging');
  }
  function moveGhost(a) {
    if(!a.ghost)return;
    a.ghost.style.left=`${a.x-a.dx}px`;
    a.ghost.style.top=`${a.y-a.dy}px`;
  }
  function rearrange(a) {
    const target=document.elementFromPoint(a.x,a.y)?.closest('.fav-card');
    if(!target||target===a.card||target.closest('.favorite-group')!==a.card.closest('.favorite-group'))return;
    const rect=target.getBoundingClientRect();
    const columns=getComputedStyle(target.parentElement).gridTemplateColumns.split(' ').length;
    const before=columns===1?a.y<rect.top+rect.height/2:a.x<rect.left+rect.width/2;
    const sibling=before?target:target.nextElementSibling;
    if(sibling===a.card||(!before&&target===a.card.previousElementSibling))return;
    const grid=a.card.parentElement;
    const cards=[...grid.querySelectorAll('.fav-card')].filter(c=>c!==a.card);
    const originals=new Map(cards.map(c=>[c,c.getBoundingClientRect()]));
    if(before)target.before(a.card);else target.after(a.card);
    if(reduced())return;
    for(const c of cards) {
      if(typeof c.animate!=='function')continue;
      const old=originals.get(c),now=c.getBoundingClientRect();
      const x=old.left-now.left,y=old.top-now.top;
      if(Math.abs(x)+Math.abs(y)>1)c.animate([{transform:`translate(${x}px,${y}px)`},{transform:'translate(0,0)'}],{duration:160,easing:'ease-out'});
    }
  }
  function autoScroll(a) {
    if(!active||active!==a||!a.changed)return;
    const edge=64;
    const dy=a.y<edge?-Math.min(16,(edge-a.y)/3):a.y>innerHeight-edge?Math.min(16,(a.y-(innerHeight-edge))/3):0;
    if(dy)window.scrollBy(0,dy);
    if(dy)rearrange(a);
    a.frame=requestAnimationFrame(()=>autoScroll(a));
  }
  container.addEventListener('pointerdown',e=>{
    const handle=e.target.closest('.drag-handle');
    const card=handle?.closest('.fav-card');
    if(!card||!container.contains(card)||e.button!==0||active)return;
    e.preventDefault();
    active={card,pointerId:e.pointerId,x:e.clientX,y:e.clientY,startX:e.clientX,startY:e.clientY,changed:false,frame:0};
    container.setPointerCapture(e.pointerId);
  });
  // Document listeners support the case where capture was lost mid-drag.
  function track(e) {
    const a=active;
    if(!a||e.pointerId!==a.pointerId)return;
    a.x=e.clientX;a.y=e.clientY;
    if(!a.changed) {
      if(Math.hypot(a.x-a.startX,a.y-a.startY)<6)return;
      a.changed=true;
      dragGhost(a);
      a.frame=requestAnimationFrame(()=>autoScroll(a));
    }
    moveGhost(a);
    rearrange(a);
  }
  function finish(e,cancelled=false){
    const a=active;
    if(!a||e.pointerId!==a.pointerId)return;
    active=null;
    cancelAnimationFrame(a.frame);
    a.ghost?.remove();
    a.card.classList.remove('dragging');
    if(container.hasPointerCapture(e.pointerId))container.releasePointerCapture(e.pointerId);
    if(cancelled){if(a.changed)onCancel();return;}
    if(a.changed)onCommit(allIDs());
  }
  // Pointer capture keeps the stream on the container even after DOM reordering.
  container.addEventListener('pointermove',track);
  document.addEventListener('pointerup',e=>finish(e));
  document.addEventListener('pointercancel',e=>finish(e,true));
  // If the window loses focus, never save a partially completed drag.
  window.addEventListener('blur',()=>{if(active)finish({pointerId:active.pointerId},true);});
  return {isDragging:()=>!!active};
}
