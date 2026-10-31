package server

import (
    "os"
    "path/filepath"
    "testing"
    "time"
)

func TestRecentTimersPersistAndDeduplicate(t *testing.T) {
    path:=filepath.Join(t.TempDir(),"templates.json")
    s,err:=NewService(path,time.Minute);if err!=nil{t.Fatal(err)}
    for _,name:=range []string{"A","B","C","D","E","F"} {
        s.StartConfigured(name,2*time.Minute,TimerOptions{Sound:true,Notify:true})
    }
    got:=s.Snapshot().Recent
    if len(got)!=5 || got[0].Name!="F" || got[4].Name!="B" {t.Fatalf("recent initial: %+v",got)}
    s.StartConfigured("D",2*time.Minute,TimerOptions{Sound:true,Notify:true})
    got=s.Snapshot().Recent
    if len(got)!=5 || got[0].Name!="D" {t.Fatalf("recent dedup: %+v",got)}
    after,err:=NewService(path,time.Minute);if err!=nil{t.Fatal(err)}
    got=after.Snapshot().Recent
    if len(got)!=5 || got[0].Name!="D" {t.Fatalf("recent after restart: %+v",got)}
    if err:=after.ClearRecent();err!=nil{t.Fatal(err)}
    if len(after.Snapshot().Recent)!=0{t.Fatal("not cleared")}
    restored,err:=NewService(path,time.Minute);if err!=nil{t.Fatal(err)}
    if len(restored.Snapshot().Recent)!=0{t.Fatal("clear not persisted")}
    if _,err:=os.Stat(path+".recent.json");err!=nil{t.Fatal(err)}
}
