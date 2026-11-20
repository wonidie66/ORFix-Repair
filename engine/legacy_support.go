package engine
import ("fmt"; "os"; "path/filepath"; "strings"; "time")
const version="4.0.0-preview"
func rel(root, path string) string {
	v, e := filepath.Rel(root, path)
	if e != nil {
		return path
	}
	return v
}
func outputLog(root string, now time.Time, content string) (string, error) {
	name := filepath.Join(root, "ORFixRepair_"+now.Format("2006-01-02_15-04-05")+".log")
	for i := 0; i < 100; i++ {
		p := name
		if i > 0 {
			p = strings.TrimSuffix(name, ".log") + fmt.Sprintf("_%d.log", i+1)
		}
		f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if os.IsExist(e) {
			continue
		}
		if e != nil {
			return "", e
		}
		_, e = f.WriteString(content)
		ce := f.Close()
		if e == nil {
			e = ce
		}
		return p, e
	}
	return "", fmt.Errorf("日志文件名冲突")
}
func reportText(r scanReport, mode string, now time.Time, preview bool, results []string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "ORFix Repair %s | %s | 扫描 %s\n", version, mode, now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&sb, "目录：%s\n扫描 INI：%d；可修改的 Mod：%d；结构诊断：%d；跳过/须检查：%d；环境警告：%d；错误：%d\n", r.root, r.scanned, len(r.repairs), len(r.findings), len(r.manual), len(r.warnings), len(r.errors))
	if preview {
		sb.WriteString("本次为预览/取消模式，没有修改任何文件。\n")
	}
	sb.WriteString("\n[建议修改]\n")
	for _, p := range r.repairs {
		fmt.Fprintf(&sb, "\n%s\n", rel(r.root, p.path))
		for _, c := range p.changes {
			fmt.Fprintf(&sb, "  + %s\n", c)
		}
	}
	if len(r.repairs) == 0 {
		sb.WriteString("无\n")
	}
	sb.WriteString("\n[执行及备份]\n")
	for _, x := range results {
		sb.WriteString(x + "\n")
	}
	if len(results) == 0 {
		sb.WriteString("无\n")
	}
	sb.WriteString("\n[面部兼容性风险（仅提醒，不自动修改）]\n")
	for _, x := range r.faceFindings {
		sb.WriteString(x + "\n")
	}
	if len(r.faceFindings) == 0 {
		sb.WriteString("未发现可由当前静态规则识别的疑似面部绑定风险；这不等于游戏渲染完全无问题。\n")
	}
	sb.WriteString("\n[结构诊断]\n")
	for _, x := range r.findings {
		sb.WriteString(x + "\n")
	}
	if len(r.findings) == 0 {
		sb.WriteString("无\n")
	}
	sb.WriteString("\n[环境检查]\n")
	for _, x := range r.warnings {
		sb.WriteString(x + "\n")
	}
	if len(r.warnings) == 0 {
		sb.WriteString("无\n")
	}
	sb.WriteString("\n[复杂情况]\n")
	for _, x := range r.manual {
		sb.WriteString(x + "\n")
	}
	if len(r.manual) == 0 {
		sb.WriteString("无\n")
	}
	sb.WriteString("\n[错误]\n")
	for _, x := range r.errors {
		sb.WriteString(x + "\n")
	}
	if len(r.errors) == 0 {
		sb.WriteString("无\n")
	}
	sb.WriteString("\n注意：需要正确安装并启用全局 ORFix。增强模式仅对已识别旧版凝光执行 Hash+Blend 联动升级；其他角色旧 Hash 暂不盲改。面部风险只提醒，不会在正常修复中自动修改；确认游戏内确有白圈/妆容异常后，再单独运行 -facefix。实验性 TexFx 网格猜测仍需手工验证。\n")
	return sb.String()
}

func confirm(_,_ string)bool{return false}
func notify(_, _ string,_ bool){}
