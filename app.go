package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"orfixrepair_v4/engine"
)

// Response is the only interface visible to React. Do not expose absolute
// backup paths for modification or user-supplied file contents as executable JS.
type Response struct {
	Phase      string   `json:"phase"`
	Root       string   `json:"root"`
	Scanned    int      `json:"scanned"`
	Modified   int      `json:"modified"`
	FaceRisks  []string `json:"faceRisks"`
	FaceManual []string `json:"faceManual"`
	Changes    []string `json:"changes"`
	Errors     []string `json:"errors"`
	LogPath    string   `json:"logPath"`
}

type App struct {
	mu       sync.Mutex
	ctx      context.Context
	root     string
	pending  bool
	normal   []engine.BatchEntry
}

func NewApp() *App { return &App{} }
func (a *App) startup(ctx context.Context) { a.ctx = ctx }

func fromStage(r engine.StageResult) Response {
	return Response{r.Phase, r.Root, r.Scanned, r.Modified,
		append([]string{}, r.FaceRisks...), append([]string{}, r.FaceManual...),
		append([]string{}, r.Changes...), append([]string{}, r.Errors...), r.LogPath}
}

func (a *App) DefaultFolder() string {
	path, err := os.Executable()
	if err != nil {return ""}
	return filepath.Dir(path)
}

func (a *App) SelectDirectory() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pending { return "", errors.New("请先完成当前面部检查，再更换目录") }
	if a.ctx == nil {return "", errors.New("Wails 窗口尚未初始化")}
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "选择 GIMI Mod 文件夹",DefaultDirectory:a.DefaultFolder()})
}

func (a *App) RunNormal(folder string) (Response,error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pending {return Response{},errors.New("请先完成待确认的面部阶段")}
	root,err:=engine.ValidateRoot(folder)
	if err!=nil{return Response{},err}
	r := engine.Normal(root,time.Now())
	a.root=root
	a.normal=append([]engine.BatchEntry{},r.RollbackItems...)
	a.pending=len(r.FaceRisks)>0 && len(r.Errors)==0
	return fromStage(r),nil
}

func (a *App) RunPreview(folder string)(Response,error) {
	a.mu.Lock();defer a.mu.Unlock()
	if a.pending{return Response{},errors.New("请先完成当前面部检查")}
	root,err:=engine.ValidateRoot(folder);if err!=nil{return Response{},err}
	return fromStage(engine.Preview(root,time.Now())),nil
}

func (a *App) RunRollback(folder string)(Response,error){
	a.mu.Lock();defer a.mu.Unlock()
	if a.pending{return Response{},errors.New("面部检查未结束，不能回滚其他目录")}
	root,err:=engine.ValidateRoot(folder);if err!=nil{return Response{},err}
	return fromStage(engine.Rollback(root,time.Now())),nil
}

// ContinueFace has NO folder argument on purpose; the GUI cannot silently
// switch directories between normal repair and user-confirmed Face Fix.
func (a *App) ContinueFace()(Response,error){
	a.mu.Lock();defer a.mu.Unlock()
	if !a.pending || strings.TrimSpace(a.root)=="" {return Response{},errors.New("当前没有等待用户确认的面部风险")}
	root:=a.root
	a.pending=false
	res:=engine.Face(root,a.normal,time.Now())
	a.normal=nil
	if len(res.Errors)==0&&res.Modified==0{
		res.Errors=append(res.Errors,"未找到可安全自动修改的面部绑定，复杂情况需要人工检查")
	}
	return fromStage(res),nil
}

// The "完成退出" action exits only the desktop window after the user decides
// not to apply Face Fix. No hidden repair task is scheduled.
func (a *App) Finish(){
	a.mu.Lock();a.pending=false;a.root="";a.normal=nil;a.mu.Unlock()
	if a.ctx!=nil { runtime.Quit(a.ctx) }
}
