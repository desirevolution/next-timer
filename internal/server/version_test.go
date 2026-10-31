package server

import (
  "net/http"
  "net/http/httptest"
  "path/filepath"
  "strings"
  "testing"
  "time"
)

func TestHTMLContainsBuildVersion(t *testing.T){
  saved:=AppVersion
  AppVersion="v9.8.7"
  defer func(){AppVersion=saved}()
  svc,err:=NewService(filepath.Join(t.TempDir(),"templates.json"),time.Minute)
  if err!=nil{t.Fatal(err)}
  for _,path:=range []string{"/","/index.html"}{
    request:=httptest.NewRequest(http.MethodGet,path,nil)
    rec:=httptest.NewRecorder()
    (API{Service:svc}).Handler().ServeHTTP(rec,request)
    if rec.Code!=200{t.Fatalf("%s status=%d",path,rec.Code)}
    if !strings.Contains(rec.Body.String(),`id="app-version" data-i18n-ignore>v9.8.7</span>`){t.Fatalf("Version not injected at %s",path)}
  }
}
