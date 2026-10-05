package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ============================================================
//  ANSI COLORS
// ============================================================
const (
	NC = "\033[0m"
	R  = "\033[1;91m"
	G  = "\033[1;92m"
	Y  = "\033[1;93m"
	U  = "\033[0;35m"
	C  = "\033[0;96m"
	W  = "\033[1;97m"
	A  = "\033[0;34m"
)

// ============================================================
//  GLOBAL STATE
// ============================================================
var (
	uiBackend string

	// system info
	prettyName    string
	ramMB         string
	uptimeStr     string
	city          string
	isp           string
	myIP          string
	domain        string
	installDate   string
	installedDays string
	version       string

	// service status
	stsWS    string
	stsNginx string
	stsXray  string
	status   string

	// account counts
	sshCount   string
	vmessCount string
	vlessCount string
	trojanCnt  string
	ssCount    string

	license = "XDTunnel Free Version"
)

// ============================================================
//  HELPERS
// ============================================================
func clearScreen() {
	fmt.Print("\033[2J\033[3J\033[H")
}

func pause(sec float64) {
	time.Sleep(time.Duration(sec * float64(time.Second)))
}

func readLine(prompt string) string {
	fmt.Print(prompt)
	r := bufio.NewReader(os.Stdin)
	s, _ := r.ReadString('\n')
	return strings.TrimSpace(s)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func fileNonEmpty(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Size() > 0
}

func readFileTrim(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func writeFile(p, content string) error {
	return os.WriteFile(p, []byte(content), 0o644)
}

func runCapture(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(out), "\n")
}

func systemctlActive(svc string) string {
	out, err := exec.Command("systemctl", "is-active", svc).Output()
	if err != nil {
		return "inactive"
	}
	return strings.TrimSpace(string(out))
}

func countLines(text string) int {
	if text == "" {
		return 0
	}
	return len(strings.Split(text, "\n"))
}

func countPatternInFile(path, pattern string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, pattern) {
			count++
		}
	}
	return count
}

// ============================================================
//  BORDERS
// ============================================================
func lineAtas()   { fmt.Printf("%s┌───────────────────────────────────────────────┐%s\n", C, NC) }
func lineBawah()  { fmt.Printf("%s└───────────────────────────────────────────────┘%s\n", C, NC) }
func lineTengah() { fmt.Printf("%s├───────────────────────────────────────────────┤%s\n", C, NC) }

// ============================================================
//  DATA COLLECTOR
// ============================================================
func collectData() {
	// license file
	if !fileNonEmpty("/etc/.license") {
		_ = writeFile("/etc/.license", license)
	}

	// install date
	if !fileExists("/root/.ins-date") {
		_ = writeFile("/root/.ins-date", time.Now().Format("2006-01-02"))
	}
	installDate = readFileTrim("/root/.ins-date")
	today := time.Now().Format("2006-01-02")

	if t1, err := time.Parse("2006-01-02", today); err == nil {
		if t2, err := time.Parse("2006-01-02", installDate); err == nil {
			installedDays = strconv.Itoa(int(t1.Sub(t2).Hours() / 24))
		}
	}
	if installedDays == "" {
		installedDays = "0"
	}

	// IP
	myIP = strings.TrimSpace(runCapture("curl", "-sS", "ipv4.icanhazip.com"))
	if myIP == "" {
		myIP = "N/A"
	}

	// city / isp
	city = readFileTrim("/root/.city")
	isp = readFileTrim("/root/.isp")
	domain = readFileTrim("/etc/xray/domain")
	version = readFileTrim("/root/.versi")

	// service status
	ws := systemctlActive("ws")
	ng := systemctlActive("nginx")
	xr := systemctlActive("xray")

	stsWS = colorStatus(ws)
	stsNginx = colorStatus(ng)
	stsXray = colorStatus(xr)

	if ws == "active" && ng == "active" && xr == "active" {
		status = G + "DONE" + NC
	} else {
		status = R + "EROR" + NC
	}

	// counts
	vlx := countPatternInFile("/etc/xray/config.json", "#& ")
	vlessCount = strconv.Itoa(vlx / 2)

	vmc := countPatternInFile("/etc/xray/config.json", "### ")
	vmessCount = strconv.Itoa(vmc / 2)

	trx := countPatternInFile("/etc/xray/config.json", "! ")
	trojanCnt = strconv.Itoa(trx / 2)

	ssx := countPatternInFile("/etc/xray/config.json", "#ss# ")
	ssCount = strconv.Itoa(ssx / 2)

	// ssh users (uid >= 1000 & name != nobody)
	sshCount = countSSHUsers()

	// os-release
	prettyName = readOSRelease()
	ramMB = readRAM()
	uptimeStr = readUptime()
}

func colorStatus(s string) string {
	if s == "active" {
		return G + "ON " + NC
	}
	return R + "OFF" + NC
}

func countSSHUsers() string {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return "0"
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}
		uid, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}
		if uid >= 1000 && parts[0] != "nobody" {
			count++
		}
	}
	return strconv.Itoa(count)
}

func readOSRelease() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			v := strings.TrimPrefix(line, "PRETTY_NAME=")
			v = strings.Trim(v, `"`)
			return v
		}
	}
	return "Linux"
}

func readRAM() string {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return "N/A"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, _ := strconv.Atoi(fields[1])
				return strconv.Itoa(kb / 1024)
			}
		}
	}
	return "N/A"
}

func readUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "N/A"
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return "N/A"
	}
	sec, _ := strconv.ParseFloat(fields[0], 64)
	d := int(sec) / 86400
	h := (int(sec) % 86400) / 3600
	m := (int(sec) % 3600) / 60
	if d > 0 {
		return fmt.Sprintf("%d days, %d hours, %d minutes", d, h, m)
	}
	return fmt.Sprintf("%d hours, %d minutes", h, m)
}

// ============================================================
//  UI BACKEND
// ============================================================
func findUI() {
	if p, err := exec.LookPath("menu-usr"); err == nil {
		uiBackend = p
		return
	}
	uiBackend = "menu-usr" // fallback
}

func callUI(args ...string) {
	if uiBackend == "" {
		fmt.Printf("%s✗ menu-usr tidak ditemukan%s\n", R, NC)
		return
	}
	cmd := exec.Command(uiBackend, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

// ============================================================
//  BANNER & INFO
// ============================================================
func banner() {
	clearScreen()
	lineAtas()
	fmt.Printf("%s│%s         %s.::.%s %sXDTunnell AUTOSCRIPT%s %s.::.%s        %s│%s\n",
		C, NC, U, NC, W, NC, U, NC, C, NC)
	lineBawah()
}

func serviceSystemOperating() {
	banner()
	lineAtas()
	fmt.Printf("%s│%s SYSTEM  : %s%s%s\n", C, NC, W, prettyName, NC)
	fmt.Printf("%s│%s RAM     : %s%s MB%s\n", C, NC, W, ramMB, NC)
	fmt.Printf("%s│%s UPTIME  : %s%s%s\n", C, NC, W, uptimeStr, NC)
	fmt.Printf("%s│%s CITY    : %s%s%s\n", C, NC, W, city, NC)
	fmt.Printf("%s│%s ISP     : %s%s%s\n", C, NC, W, isp, NC)
	fmt.Printf("%s│%s IP      : %s%s%s\n", C, NC, W, myIP, NC)
	fmt.Printf("%s│%s DOMAIN  : %s%s%s\n", C, NC, W, domain, NC)
	lineBawah()

	lineAtas()
	fmt.Printf("%s│%s%s SSHWS :%s %s %s%s│%s%s NGINX :%s %s %s%s│%s%s XRAY :%s %s %s%s│%s %s\n",
		C, NC, W, NC, stsWS, NC, C, NC, W, NC, stsNginx, NC, C, NC, W, NC, stsXray, NC, C, NC, status)
	lineBawah()
}

func listAllAccount() {
	lineAtas()
	fmt.Printf("%s│%s %s               TOTAL ACCOUNTS                 %s│%s\n", C, NC, W, C, NC)
	lineTengah()
	fmt.Printf("%s│%s  SSHWS   : %s%-6s%s    VMESS  : %s%-6s%s   %s %s\n",
		C, NC, G, sshCount, NC, G, vmessCount, NC, C, NC)
	fmt.Printf("%s│%s  VLESS   : %s%-6s%s    TROJAN : %s%-6s%s   %s %s\n",
		C, NC, G, vlessCount, NC, G, trojanCnt, NC, C, NC)
	fmt.Printf("%s│%s  SHADOWSOCKS : %s%-6s%s                   %s %s\n",
		C, NC, G, ssCount, NC, C, NC)
	lineBawah()
}

func detailsClientsName() {
	lineAtas()
	fmt.Printf("%s│%s  VERSION      : %s%s%s\n", C, NC, W, version, NC)
	fmt.Printf("%s│%s  CLIENTS      : %s%s%s\n", C, NC, W, license, NC)
	fmt.Printf("%s│%s  INSTALL TIME : %s%s Day(s) (%s)%s\n",
		C, NC, W, installedDays, installDate, NC)
	lineBawah()
}

// ============================================================
//  COMMAND / SELECTOR
// ============================================================
func command() {
	lineAtas()
	fmt.Printf("%s│ %s1.)%s☞ %s SSH/OpenVPN        %s6.)%s☞ %s UPDATE SCRIPT  %s│%s\n",
		C, A, Y, W, A, Y, W, C, NC)
	fmt.Printf("%s│ %s2.)%s☞ %s XRAY VMESS         %s7.)%s☞ %s BACKUP RESTORE %s│%s\n",
		C, A, Y, W, A, Y, W, C, NC)
	fmt.Printf("%s│ %s3.)%s☞ %s XRAY VLESS         %s8.)%s☞ %s FEATURES       %s│%s\n",
		C, A, Y, W, A, Y, W, C, NC)
	fmt.Printf("%s│ %s4.)%s☞ %s XRAY TROJAN        %s9.)%s☞ %s REBOOT         %s│%s\n",
		C, A, Y, W, A, Y, W, C, NC)
	fmt.Printf("%s│ %s5.)%s☞ %s XRAY SHADOWSOCKS  %s10.)%s☞ %s MENU BOT       %s│%s\n",
		C, A, Y, W, A, Y, W, C, NC)
	lineBawah()
}

func accesUseCommand() {
	fmt.Println()
	fmt.Printf("%s —————————————————————————————————————————————————%s\n", C, NC)
	fmt.Printf("%s             acces use%s %s☞%s %smenu%s %s enter%s\n",
		C, NC, Y, NC, G, NC, C, NC)
	fmt.Printf("%s —————————————————————————————————————————————————%s\n", C, NC)
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	mainMenu()
}

func selectDisplay() {
	fmt.Println()
	choice := readLine(" Select options [ 1 - 10 ] : ")
	switch choice {
	case "1":
		clearScreen()
		menuSSH()
	case "2":
		clearScreen()
		menuVMess()
	case "3":
		clearScreen()
		menuVLESS()
	case "4":
		clearScreen()
		menuTrojan()
	case "5":
		clearScreen()
		menuShadowsocks()
	case "6":
		clearScreen()
		updateScript()
	case "7":
		clearScreen()
		runCapture("m-bkp")
	case "8":
		clearScreen()
		runCapture("m-ftr")
	case "9":
		clearScreen()
		_ = exec.Command("reboot").Run()
	case "10":
		clearScreen()
		runCapture("m-bot")
	default:
		mainMenu()
	}
}

func updateScript() {
	hosting := "https://raw.githubusercontent.com/g9ktx7lm/my/main/"
	_ = os.MkdirAll("/cache", 0o755)
	_ = os.MkdirAll("/usr/local/style", 0o755)
	if err := os.Chdir("/cache"); err != nil {
		fmt.Println("chdir error:", err)
		return
	}
	_ = exec.Command("wget", "-q", "-O", "menu.zip", hosting+"v1/menu.zip").Run()
	_ = exec.Command("7z", "x", "-pmwhehehehhe", "menu.zip").Run()
	_ = exec.Command("chmod", "+x", "menu").Run()

	// move *.sh → /usr/local/style/
	_ = filepath.Walk("menu", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".sh") {
			_ = os.Rename(path, "/usr/local/style/"+filepath.Base(path))
		} else {
			_ = os.Rename(path, "/usr/local/sbin/"+filepath.Base(path))
		}
		return nil
	})

	_ = os.Chdir("/root")
	_ = os.RemoveAll("/cache")
	_ = exec.Command("wget", "-qO-", hosting+"version").Run()

	// write version manually since wget output we don't capture here
	if out, err := exec.Command("wget", "-qO-", hosting+"version").Output(); err == nil {
		_ = writeFile("/root/.versi", strings.TrimSpace(string(out)))
	}

	fmt.Println("Done ✓")
	pause(2)
	mainMenu()
}

// ============================================================
//  SUB-MENUS
// ============================================================
func menuSSH() {
	for {
		clearScreen()
		lineAtas()
		fmt.Printf("%s│ %s               SSH - OPENVPN                  %s│%s\n", C, W, C, NC)
		lineBawah()
		lineAtas()
		fmt.Printf("%s│%s  %s── ACCOUNT ──%s\n", C, NC, U, NC)
		fmt.Printf("%s│%s  1. %sCheck Users Login%s\n", C, NC, W, NC)
		fmt.Printf("%s│%s  2. %sCreate Account%s\n", C, NC, W, NC)
		fmt.Printf("%s│%s  3. %sDelete Account%s\n", C, NC, W, NC)
		fmt.Printf("%s│%s  4. %sRenew Account%s\n", C, NC, W, NC)
		fmt.Printf("%s│%s  5. %sTrial Account%s\n", C, NC, W, NC)
		lineTengah()
		fmt.Printf("%s│%s  %s── MEMBER ──%s\n", C, NC, U, NC)
		fmt.Printf("%s│%s  6. %sList Member Accounts%s\n", C, NC, W, NC)
		fmt.Printf("%s│%s  7. %sList Expired Accounts%s\n", C, NC, W, NC)
		lineTengah()
		fmt.Printf("%s│%s  %s── LOCK & UNLOCK ──%s\n", C, NC, U, NC)
		fmt.Printf("%s│%s  8. %sLock Account%s\n", C, NC, W, NC)
		fmt.Printf("%s│%s  9. %sUnlock Account%s\n", C, NC, W, NC)
		lineTengah()
		fmt.Printf("%s│%s  %s── LIMIT ──%s\n", C, NC, U, NC)
		fmt.Printf("%s│%s 10. %sEdit Limit IP%s\n", C, NC, W, NC)
		lineTengah()
		fmt.Printf("%s│%s  %s── OTHER ──%s\n", C, NC, U, NC)
		fmt.Printf("%s│%s 11. %sDetail Account%s\n", C, NC, W, NC)
		fmt.Printf("%s│%s 12. %sRecovery Account%s\n", C, NC, W, NC)
		lineBawah()
		lineAtas()
		fmt.Printf("%s│%s 13. %sBack to Menu%s\n", C, NC, R, NC)
		fmt.Printf("%s│%s  x. %sExit%s\n", C, NC, R, NC)
		lineBawah()
		fmt.Println()

		opt := readLine(" Select Options [ 1 - 13 or x ] : ")
		switch opt {
		case "1":
			clearScreen()
			callUI("cek_login", "ssh")
		case "2":
			clearScreen()
			callUI("add", "ssh")
		case "3":
			clearScreen()
			callUI("delete", "ssh")
		case "4":
			clearScreen()
			callUI("renew", "ssh")
		case "5":
			clearScreen()
			callUI("trial", "ssh")
		case "6":
			clearScreen()
			callUI("member", "ssh")
		case "7":
			clearScreen()
			fmt.Printf("  %sComing Soon%s\n", R, NC)
			pause(2)
		case "8":
			clearScreen()
			callUI("lock", "ssh")
		case "9":
			clearScreen()
			callUI("unlock", "ssh")
		case "10":
			clearScreen()
			callUI("edit_limit", "ssh")
		case "11":
			clearScreen()
			callUI("cek_config", "ssh")
		case "12":
			clearScreen()
			callUI("recovery", "ssh")
		case "13":
			clearScreen()
			return
		case "x":
			os.Exit(0)
		}
	}
}

func xrayMenu(title, proto string, withCustom bool) {
	for {
		clearScreen()
		lineAtas()
		fmt.Printf("%s│ %s%s%s│%s\n", C, W, centerPad(title, 44), C, NC)
		lineBawah()
		lineAtas()
		fmt.Printf("%s│%s  %s── ACCOUNT ──%s\n", C, NC, U, NC)
		fmt.Printf("%s│%s  1. %sCheck Users Login%s\n", C, NC, W, NC)
		fmt.Printf("%s│%s  2. %sList Member%s\n", C, NC, W, NC)
		fmt.Printf("%s│%s  3. %sCreate Account%s\n", C, NC, W, NC)
		if withCustom {
			fmt.Printf("%s│%s  4. %sCreate Account %s[custom secret]%s\n", C, NC, W, U, NC)
		}
		fmt.Printf("%s│%s  5. %sTrial Account%s\n", C, NC, W, NC)
		fmt.Printf("%s│%s  6. %sDelete Account%s\n", C, NC, W, NC)
		fmt.Printf("%s│%s  7. %sRenew Account%s\n", C, NC, W, NC)
		lineTengah()
		fmt.Printf("%s│%s  %s── CONFIG ──%s\n", C, NC, U, NC)
		fmt.Printf("%s│%s  8. %sCheck Config%s\n", C, NC, W, NC)
		fmt.Printf("%s│%s  9. %sRecovery Account%s\n", C, NC, W, NC)
		lineTengah()
		fmt.Printf("%s│%s  %s── LIMIT ──%s\n", C, NC, U, NC)
		fmt.Printf("%s│%s 10. %sEdit Limit IP%s\n", C, NC, W, NC)
		fmt.Printf("%s│%s 11. %sEdit Limit Bandwidth%s\n", C, NC, W, NC)
		lineTengah()
		fmt.Printf("%s│%s  %s── LOCK & UNLOCK ──%s\n", C, NC, U, NC)
		fmt.Printf("%s│%s 12. %sLock Account%s\n", C, NC, W, NC)
		fmt.Printf("%s│%s 13. %sUnlock Account%s\n", C, NC, W, NC)
		lineTengah()
		fmt.Printf("%s│%s  %s── OTHER ──%s\n", C, NC, U, NC)
		label := "Change UUID"
		if proto == "trojan" || proto == "shadowsocks" {
			label = "Change Password"
		}
		fmt.Printf("%s│%s 14. %s%s%s\n", C, NC, W, label, NC)
		lineBawah()
		lineAtas()
		fmt.Printf("%s│%s 15. %sBack to Menu%s\n", C, NC, R, NC)
		fmt.Printf("%s│%s  x. %sExit%s\n", C, NC, R, NC)
		lineBawah()
		fmt.Println()

		opt := readLine(" Select Options [ 1 - 15 or x ] : ")
		switch opt {
		case "1":
			clearScreen()
			callUI("cek_login", proto)
		case "2":
			clearScreen()
			callUI("member", proto)
		case "3":
			clearScreen()
			callUI("add", proto)
		case "4":
			if withCustom {
				clearScreen()
				callUI("custom_uuid", proto)
			}
		case "5":
			clearScreen()
			callUI("trial", proto)
		case "6":
			clearScreen()
			callUI("delete", proto)
		case "7":
			clearScreen()
			callUI("renew", proto)
		case "8":
			clearScreen()
			callUI("cek_config", proto)
		case "9":
			clearScreen()
			callUI("recovery", proto)
		case "10":
			clearScreen()
			callUI("edit_limit", proto)
		case "11":
			clearScreen()
			callUI("edit_quota", proto)
		case "12":
			clearScreen()
			callUI("lock", proto)
		case "13":
			clearScreen()
			callUI("unlock", proto)
		case "14":
			clearScreen()
			callUI("ganti_uuid", proto)
		case "15":
			clearScreen()
			return
		case "x":
			os.Exit(0)
		}
	}
}

func menuVMess()       { xrayMenu("XRAY - VMESS", "vmess", true) }
func menuVLESS()       { xrayMenu("XRAY - VLESS", "vless", true) }
func menuTrojan()      { xrayMenu("XRAY - TROJAN", "trojan", true) }
func menuShadowsocks() { xrayMenu("XRAY - SHADOWSOCKS", "shadowsocks", true) }

func centerPad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	total := width - len(s)
	left := total / 2
	right := total - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// ============================================================
//  MAIN
// ============================================================
func mainMenu() {
	collectData()
	serviceSystemOperating()
	listAllAccount()
	pause(0.5)
	command()
	selectDisplay()
}

func main() {
	findUI()

	// Subcommand routing
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "welcome", "xdxl":
			collectData()
			serviceSystemOperating()
			listAllAccount()
			pause(0.5)
			detailsClientsName()
			accesUseCommand()
			return
		case "ssh":
			menuSSH()
			return
		case "vmess":
			menuVMess()
			return
		case "vless":
			menuVLESS()
			return
		case "trojan":
			menuTrojan()
			return
		case "ss", "shadowsocks":
			menuShadowsocks()
			return
		}
	}

	// Default
	mainMenu()
}

// pastikan runtime dipakai (untuk build tag kalau nanti mau cek GOOS)
var _ = runtime.GOOS
