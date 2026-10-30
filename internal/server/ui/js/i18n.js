/* NextTimer interface localization. German is the source language for UI copy;
 * timer/favorite/group names remain user data and are never translated.
 * Uses no dependencies, works in the server and standalone PWA editions. */
(() => {
  'use strict';
  const KEY = 'nexttimer.language.v1';
  const translations = {
    // Shared shell / quick start
    'NextTimer auf GitHub öffnen':'Open NextTimer on GitHub',
    'GitHub-Repository öffnen':'Open GitHub repository',
    'Verbinde …':'Connecting …','Verbunden':'Connected','Verbinde erneut …':'Reconnecting …',
    'Tastaturkürzel anzeigen':'Show keyboard shortcuts','Tastaturkürzel (?)':'Keyboard shortcuts (?)',
    'Einstellungen öffnen':'Open settings','Einstellungen':'Settings',
    'Eine neue Version von NextTimer ist verfügbar.':'A new NextTimer version is available.',
    'Aktualisieren, wenn du bereit bist. Laufende lokale Timer bleiben gespeichert.':'Update when you are ready. Running local timers will be preserved.',
    'Später':'Later','Aktualisieren':'Update','Aktualisiere …':'Updating …',
    'Abgelaufene Timer':'Expired timers','Timer abgelaufen':'Timer expired','Alle bestätigen':'Acknowledge all',
    'Abgelaufen':'Expired','Bestätigen':'Acknowledge',
    'Neuer Timer':'New timer','z. B. 5m Tee oder #1':'e.g. 5m Tea or #1',
    'Dauer und optionaler Name':'Duration and optional name','▶ Start':'▶ Start',
    'Dauer vorgeben':'Set duration','Optionen':'Options','Timer':'Timer',
    'Nur in diesem Browser':'Only in this browser','Ohne Server, AWTRIX und CLI':'Without server, AWTRIX or CLI',
    'Browser-Alarm':'Browser alarm','Browser-Alarme':'Browser alarms',
    'Eigene Alarmeinstellungen':'Custom alarm settings','Nur für den nächsten Timerstart':'Only for the next timer',
    'Ton':'Sound','Ton anhören':'Preview sound','Kein Ton':'No sound','Wiederholung':'Repeat',
    'Einmal':'Once','Dreimal':'Three times','Bis zur Bestätigung':'Until acknowledged',
    'Lautstärke':'Volume','Rote Hervorhebung':'Red highlight','Systembenachrichtigung':'System notification',
    // Focus / timer actions
    'Bereit':'Ready','Pausiert':'Paused','Läuft':'Running','Lokal · läuft':'Local · running',
    'Noch kein Timer aktiv':'No active timer','Nur in diesem Browser · nicht auf AWTRIX':'Only in this browser · not on AWTRIX',
    'Vorherigen Timer anzeigen':'Show previous timer','Nächsten Timer anzeigen':'Show next timer',
    'Vorheriger Timer (←)':'Previous timer (←)','Nächster Timer (→)':'Next timer (→)',
    'Fortschritt des Haupttimers':'Main timer progress',
    '1 Min.':'1 min','Pause':'Pause','Weiter':'Resume','■ Timer beenden':'■ Stop timer',
    '☆ Favorit':'☆ Favorite','Als Favorit speichern':'Save as favorite','Timer beenden':'Stop timer',
    'Timer beenden?':'Stop timer?','Diesen Timer groß anzeigen':'Show this timer in the main view',
    'Pausieren':'Pause','Fortsetzen':'Resume','Eine Minute hinzufügen':'Add one minute',
    'Keine weiteren Timer.':'No other timers.','Weitere Timer':'Other timers',
    'Anzeigen':'Show','Lokal':'Local','Nur Browser':'Browser only',
    // Favorites and groups
    'Favoriten':'Favorites','＋ Favorit':'＋ Favorite','✎ Bearbeiten':'✎ Edit',
    '✎ Favoriten bearbeiten':'✎ Edit favorites','✓ Fertig':'✓ Done',
    'Favoriten bearbeiten, löschen oder innerhalb ihrer Gruppe umsortieren. Die Reihenfolge wird automatisch gespeichert.':'Edit, delete or reorder favorites within their groups. The order is saved automatically.',
    'Vorlagen bearbeiten und innerhalb der Gruppen sortieren':'Edit templates and reorder within groups',
    'Noch keine Favoriten. Lege deinen ersten Favoriten an.':'No favorites yet. Create your first favorite.',
    'Standard':'Default','Favorit hinzufügen':'Add favorite',
    'Gruppe nach oben':'Move group up','Gruppe nach unten':'Move group down',
    '✎ Umbenennen':'✎ Rename','Favorit bearbeiten':'Edit favorite',
    'Favorit löschen':'Delete favorite','Favorit löschen?':'Delete favorite?',
    'Innerhalb der Gruppe zurück':'Move up in group','Innerhalb der Gruppe vor':'Move down in group',
    'In der Gruppe verschieben':'Move within group','Favorit starten':'Start favorite',
    'Neuer Favorit':'New favorite','Name':'Name','z. B. Kaffee':'e.g. Coffee',
    'Dauer':'Duration','z. B. 5m oder 1h 30m':'e.g. 5m or 1h 30m',
    'Schnellstart (optional)':'Quick launch (optional)','Keine Nummer':'No number',
    '#Nummer + Enter startet den Favoriten direkt.':'#number + Enter starts the favorite directly.',
    'Gruppe (optional)':'Group (optional)','Bestehende Gruppen auswählen':'Choose an existing group',
    'Globale Einstellungen':'Global settings',
    'Sonst gelten die Einstellungen dieses Browsers.':'Otherwise, this browser’s settings apply.',
    'Weitere Optionen':'More options','Abbrechen':'Cancel','Favorit speichern':'Save favorite',
    'Änderungen speichern':'Save changes','Schließen':'Close','Gruppe umbenennen':'Rename group',
    'Alle Favoriten dieser Gruppe erhalten den neuen Gruppennamen.':'All favorites in this group will get the new group name.',
    'Neuer Name':'New name','Umbenennen':'Rename','Löschen':'Delete',
    // Settings and backups
    'Standardwerte für diesen Browser. Favoriten können eigene Einstellungen haben.':'Defaults for this browser. Favorites can override them.',
    'Benachrichtigungen aktivieren':'Enable notifications',
    '✓ Benachrichtigungen aktiv':'✓ Notifications enabled',
    'Benachrichtigungen blockiert · Freigabe in den Browser-Einstellungen möglich':'Notifications blocked · enable them in browser settings',
    'Systembenachrichtigungen hier nicht verfügbar':'System notifications are unavailable here',
    'Töne benötigen eine geöffnete App. Systembenachrichtigungen benötigen eine Browserfreigabe.':'Sounds require the app to be open. System notifications require browser permission.',
    'Daten & Sicherung':'Data & backup',
    'Favoriten und lokale Timer liegen nur in diesem Browser.':'Favorites and local timers are stored in this browser only.',
    'JSON-Backup herunterladen':'Download JSON backup','JSON-Backup importieren':'Import JSON backup',
    'Sprache':'Language','Automatisch':'Automatic','Deutsch':'Deutsch','Englisch':'English',
    'Sprache der Oberfläche':'Interface language',
    'Automatisch verwendet die Browsersprache. Diese Einstellung gilt nur für diesen Browser.':'Automatic follows your browser language. This setting applies only to this browser.',
    'App & Offline':'App & offline','PWA-Update, Offline-Betrieb und Reparatur.':'PWA updates, offline mode and troubleshooting.',
    'PWA-Cache reparieren':'Repair PWA cache',
    // Confirmations / shortcut help
    'Wiederherstellen':'Restore','Sicherung wiederherstellen?':'Restore backup?',
    'Alle vorhandenen lokalen Favoriten und Timer werden durch diese Sicherung ersetzt.':'This backup will replace all existing local favorites and timers.',
    'Bestätigen':'Confirm','← / → Auswahl · Enter: ausgewählte Aktion · Esc: abbrechen':'← / → select · Enter: activate selected action · Esc: cancel',
    'Tastatursteuerung':'Keyboard shortcuts','Timer-Eingabe fokussieren':'Focus timer input',
    'Timer oder Favorit starten':'Start timer or favorite',
    'Favorit über die Eingabe starten':'Start favorite from the input',
    'Eingabe leeren; sonst Timer auswählen / zur Eingabe zurück':'Clear input; otherwise select a timer / return to input',
    'Zwischen angezeigten Timern wechseln':'Switch between displayed timers',
    'Angezeigten Timer pausieren/fortsetzen':'Pause/resume displayed timer',
    'Angezeigten Timer beenden (mit Bestätigung)':'Stop displayed timer (with confirmation)',
    'Diese Hilfe öffnen':'Open this help',
    'Buchstaben-Shortcuts funktionieren außerhalb von Textfeldern. Esc funktioniert überall außerhalb von Dialogen; Tab navigiert weiterhin zwischen Bedienelementen. In Bestätigungsdialogen wählt ← / → die Aktion.':
      'Letter shortcuts work outside text fields. Esc works anywhere outside dialogs; Tab continues to navigate controls. In confirmation dialogs, ← / → selects an action.',
    // Dynamic status and validation (including errors from browser-only storage)
    'Speichere Reihenfolge …':'Saving order …','Reihenfolge gespeichert.':'Order saved.',
    'Speichere Gruppenreihenfolge …':'Saving group order …','Gruppenreihenfolge gespeichert.':'Group order saved.',
    'Gruppe umbenannt.':'Group renamed.','Sicherung importiert.':'Backup imported.',
    'Favorit nicht gefunden.':'Favorite not found.',
    'Favorit mit #1 bis #0 aufrufen.':'Use #1 through #0 to launch a favorite.',
    'Bitte einen Namen eingeben.':'Please enter a name.',
    'Bitte eine Dauer eingeben.':'Please enter a duration.',
    'Bitte die Dauer getrennt vom Namen eingeben.':'Please enter the duration separately from the name.',
    'Server nicht erreichbar. Lokale Timer funktionieren offline; gespeicherte Favoriten können lokal gestartet werden.':
      'Server unavailable. Local timers still work offline; cached favorites can be started locally.',
    'Dauer angeben, z. B. 5m Tee.':'Enter a duration, e.g. 5m Tea.',
    'Dauer und Name mit einem Leerzeichen trennen, z. B. 5m Tee.':'Separate duration and name with a space, e.g. 5m Tea.',
    'Name zu lang (maximal 80 Zeichen).':'Name too long (maximum 80 characters).',
    'Dauer angeben, z. B. 10m oder 1h30m.':'Enter a duration, e.g. 10m or 1h30m.',
    'Dauer muss zwischen 1s und 168h liegen.':'Duration must be between 1s and 168h.',
    'Lokale Timer können nicht gespeichert werden (Browserspeicher voll oder gesperrt).':'Local timers could not be saved (browser storage full or blocked).',
    'Lokaler Timer nicht gefunden.':'Local timer not found.',
    'Maximale Restzeit: 168h.':'Maximum remaining time: 168h.',
    'Aktion für den aktuellen Zustand nicht möglich.':'This action is not possible in the current state.',
    'Ungültiger Gruppenname (maximal 48 Zeichen).':'Invalid group name (maximum 48 characters).',
    'Ungültige Alarmeinstellungen.':'Invalid alarm settings.',
    'Ungültige Browser-Alarmeinstellungen.':'Invalid browser alarm settings.',
    'Lautstärke muss zwischen 0 und 100 % liegen.':'Volume must be between 0 and 100%.',
    'Favoritenname muss 1–80 Zeichen haben.':'Favorite name must have 1–80 characters.',
    'Schnellstart muss eine Ziffer von 0 bis 9 sein.':'Quick launch must be a digit from 0 to 9.',
    'Ungültige Favoritendaten.':'Invalid favorite data.',
    'Doppelte oder ungültige Favoriten-ID.':'Duplicate or invalid favorite ID.',
    'Diese Schnellstart-Nummer ist bereits vergeben.':'This quick-launch number is already assigned.',
    'Name in dieser Gruppe bereits vergeben.':'The name is already used in this group.',
    'Favoriten konnten nicht gespeichert werden. Prüfe den Browserspeicher.':'Could not save favorites. Check browser storage.',
    'Favoritenliste wurde verändert.':'The favorites list has changed.',
    'Favoriten lassen sich nur innerhalb einer Gruppe verschieben.':'Favorites can only be moved within a group.',
    'Gruppenliste wurde verändert.':'The group list has changed.',
    'Gruppe nicht gefunden.':'Group not found.',
    'Gruppenname bereits vergeben.':'Group name already in use.',
    'Dieser Befehl ist im Browser-Modus nicht verfügbar.':'This command is unavailable in browser-only mode.',
    'Keine gültige NextTimer-Standalone-Sicherung.':'Not a valid NextTimer standalone backup.',
    'Ungültige Timerdaten.':'Invalid timer data.',
    'Systembenachrichtigungen benötigen HTTPS oder localhost und einen unterstützten Browser.':
      'System notifications require HTTPS or localhost and a supported browser.',
    'Der Countdown ist abgelaufen.':'The countdown has finished.',
    'Fertig!':'Finished!',
    '„{name}“ ({time} verbleibend) wirklich beenden?':'Really stop “{name}” ({time} remaining)?',
    '„{name}“ wird dauerhaft gelöscht.':'“{name}” will be permanently deleted.',
    'Mit #{number} in der Eingabe starten':'Start from input using #{number}',
    'Eine neue Version von NextTimer ist verfügbar.':'A new version of NextTimer is available.',
    // Repair page
    'NextTimer – PWA reparieren':'NextTimer – Repair PWA',
    'Entfernt nur den Offline-Cache und den Service Worker dieser Installation. Deine lokalen Timer und Favoriten bleiben gespeichert.':
      'Removes only the offline cache and service worker for this installation. Your local timers and favorites remain saved.',
    'PWA-Cache zurücksetzen':'Reset PWA cache',
  };
  const reverse = new Map(Object.entries(translations).map(([de,en])=>[en,de]));
  const variable = [
    [/^(\d+) von (\d+)$/,(_,a,b)=>`${a} of ${b}`,/^\d+ of \d+$/],
    [/^Kein Favorit für (#[0-9]) zugewiesen\.$/,(_,v)=>`No favorite assigned to ${v}.`,/^No favorite assigned to #[0-9]\.$/],
    [/^Schnellstart (#[0-9]) ist bereits vergeben\.$/,(_,v)=>`Quick launch ${v} is already assigned.`,/^Quick launch #[0-9] is already assigned\.$/],
    [/^(\d+) Timer fertig!$/,(_,n)=>`${n} timers finished!`,/^\d+ timers finished!$/],
  ];
  const modeValues = new Set(['auto','de','en']);
  const getMode = () => { try {const v=localStorage.getItem(KEY);return modeValues.has(v)?v:'auto';} catch{return 'auto';} };
  let mode=getMode();
  const resolve = () => mode==='auto' ? (/^de(?:-|$)/i.test(navigator.language||'')?'de':'en') : mode;
  let language=resolve();
  const shouldSkip = node => node.parentElement?.closest('[data-i18n-ignore],script,style,textarea,[contenteditable="true"]');
  function convert(value) {
    const raw=String(value);
    const stripped=raw.trim();
    if (!stripped) return raw;
    let result=language==='en'?translations[stripped]:reverse.get(stripped);
    if (result===undefined) {
      // Dynamic labels generated by the timer view / validation.
      const numberMatch=stripped.match(/^(\d+) von (\d+)$/);
      if(language==='en'&&numberMatch) result=`${numberMatch[1]} of ${numberMatch[2]}`;
      const englishMatch=stripped.match(/^(\d+) of (\d+)$/);
      if(language==='de'&&englishMatch) result=`${englishMatch[1]} von ${englishMatch[2]}`;
      const m=stripped.match(/^Kein Favorit für (#[0-9]) zugewiesen\.$/);
      if(language==='en'&&m) result=`No favorite assigned to ${m[1]}.`;
      const m2=stripped.match(/^No favorite assigned to (#[0-9])\.$/);
      if(language==='de'&&m2) result=`Kein Favorit für ${m2[1]} zugewiesen.`;
      const m3=stripped.match(/^Schnellstart (#[0-9]) ist bereits vergeben\.$/);
      if(language==='en'&&m3) result=`Quick launch ${m3[1]} is already assigned.`;
      const m4=stripped.match(/^Quick launch (#[0-9]) is already assigned\.$/);
      if(language==='de'&&m4) result=`Schnellstart ${m4[1]} ist bereits vergeben.`;
      const m5=stripped.match(/^(\d+) Timer fertig!$/);
      if(language==='en'&&m5) result=`${m5[1]} timers finished!`;
      const m6=stripped.match(/^(\d+) timers finished!$/);
      if(language==='de'&&m6) result=`${m6[1]} Timer fertig!`;
    }
    if(result===undefined||result===stripped)return raw;
    return raw.slice(0,raw.indexOf(stripped))+result+raw.slice(raw.indexOf(stripped)+stripped.length);
  }
  function translateNode(node) {
    if(node.nodeType===Node.TEXT_NODE) {
      if(shouldSkip(node))return;
      const updated=convert(node.nodeValue);
      if(updated!==node.nodeValue)node.nodeValue=updated;
      return;
    }
    if(node.nodeType!==Node.ELEMENT_NODE || node.closest('[data-i18n-ignore]'))return;
    for(const attr of ['placeholder','title','aria-label']) {
      if(node.hasAttribute(attr)){
        const value=node.getAttribute(attr), updated=convert(value);
        if(value!==updated)node.setAttribute(attr,updated);
      }
    }
    for(const child of node.childNodes)translateNode(child);
  }
  function translateDocument(){
    document.documentElement.lang=language;
    translateNode(document.body);
    if(document.title) document.title=convert(document.title);
    const selector=document.getElementById('language-select');
    if(selector && selector.value!==mode)selector.value=mode;
  }
  function setMode(next,save=true) {
    if(!modeValues.has(next))return;
    mode=next;
    language=resolve();
    if(save) try {localStorage.setItem(KEY,mode);} catch {}
    translateDocument();
  }
  const t = (de,params={}) => {
    const base=convert(de);
    return base.replace(/\{(\w+)\}/g,(_,key)=>String(params[key]??''));
  };
  const observer=new MutationObserver(changes=>{
    for(const entry of changes){
      if(entry.type==='characterData')translateNode(entry.target);
      else if(entry.type==='attributes')translateNode(entry.target);
      else for(const node of entry.addedNodes)translateNode(node);
    }
  });
  function init(){
    translateDocument();
    document.getElementById('language-select')?.addEventListener('change',event=>setMode(event.target.value));
    observer.observe(document.body,{subtree:true,childList:true,characterData:true,attributes:true,attributeFilter:['title','placeholder','aria-label']});
    window.addEventListener('storage',event=>{if(event.key===KEY)setMode(getMode(),false);});
    window.addEventListener('languagechange',()=>{if(mode==='auto')setMode('auto',false);});
  }
  window.NextTimerI18n={t,setMode,getMode:()=>mode,getLanguage:()=>language,translateDocument};
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',init,{once:true});
  else init();
})();
