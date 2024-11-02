package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-ping/ping"
)

var chans chan struct{}
var wg sync.WaitGroup
var ips []string
var netDetails []map[string]string
var ouiDB map[string]string
var lineNumber int
var lock *sync.RWMutex

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	chans = make(chan struct{}, 3000)
	ouiDB, _ = loadOUIDatabase("oui.csv")
	lock = &sync.RWMutex{}

	fmt.Println("------------------------------------------------------------")
	fmt.Println("行数\t状态\t\t名称\t\t\t\tIP\t\t\tMac\t\t\t制造商\n")

	// 获取主机的IP地址和子网掩码
	localIPs, err := getLocalIPs()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, ipDetail := range localIPs {
		ip := ipDetail["ip"]
		subnetMask := ipDetail["subnetMask"]

		ipRange, err := getIPRange(ip, subnetMask)
		if err != nil {
			log.Printf("获取IP范围出错: %s\n", err)
			continue
		}
		ips = append(ips, ipRange...)
	}

	scanDevices(ips)

	wg.Wait()
}

// 获取主机的IP地址和子网掩码
func getLocalIPs() ([]map[string]string, error) {
	var details []map[string]string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("获取网络接口失败: %v", err)
	}

	for _, iface := range ifaces {
		// 判断网卡是否启动
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			fmt.Println("获取地址失败:", err)
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)

			if ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				// fmt.Println(iface.Name, ipnet.IP, net.IP(ipnet.Mask).String())
				detail := map[string]string{
					"name":       iface.Name,
					"ip":         ipnet.IP.String(),
					"subnetMask": net.IP(ipnet.Mask).String(),
				}
				details = append(details, detail)
			}
		}
	}
	return details, nil
}

func getIPRange(ip, subnetMask string) ([]string, error) {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return nil, fmt.Errorf("无效的IP地址: %s", ip)
	}

	mask := net.IPMask(net.ParseIP(subnetMask).To4())
	if mask == nil {
		return nil, fmt.Errorf("无效的子网掩码: %s", subnetMask)
	}
	network := ipAddr.Mask(mask)
	broadcast := net.IP(make([]byte, 4))
	for i := 0; i < 4; i++ {
		broadcast[i] = network[i] | ^mask[i]
	}

	var ipRange []string

	for ip := network; !ip.Equal(broadcast); incrementIP(ip) {
		ipRange = append(ipRange, ip.String())
	}
	ipRange = append(ipRange, broadcast.String())
	return ipRange, nil
}

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// 扫描网段内的设备
func scanDevices(ipRange []string) {

	// 使用并发安全的 map 来存储已处理的 IP 地址
	processedIPs := sync.Map{}

	for _, ip := range ipRange {
		chans <- struct{}{}
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			defer func() {
				<-chans
			}()
			// 检查 IP 地址是否已处理
			if _, loaded := processedIPs.LoadOrStore(ip, struct{}{}); loaded {
				return
			}

			var hostNames []string

			hostNames, err := net.LookupAddr(ip)
			if err != nil {
				hostNames = []string{ip}
			}

			status := getIPStatus(ip)
			if status == "在线" || status == "离线" {
				mac, _ := getMACAddress(ip)
				manufacturer := getManufacturer(mac)
				lock.Lock()
				lineNumber++
				fmt.Printf("%v\t%s\t\t%s\t\t%s\t\t%s\t\t%s\n", lineNumber, status, hostNames[0], ip, mac, manufacturer)
				lock.Unlock()

			}
		}(ip)
	}
}

// 使用 ping 库获取 IP 地址的状态

func getIPStatus(ip string) string {
	pinger, err := ping.NewPinger(ip)

	if err != nil {
		return "Inactive"
	}
	pinger.Count = 1
	pinger.Timeout = time.Second * 1
	pinger.SetPrivileged(true)

	err = pinger.Run() // Blocks until finished.
	if err != nil {
		return "Inactive"
	}
	stats := pinger.Statistics()
	if stats.PacketsRecv > 0 {
		return "在线"
	}

	if stats.PacketsRecv == 0 {
		return "Inactive"
	}

	return "离线"
}

// func getIPStatus(ip string) string {
// 	out, err := exec.Command("ping w 1000 n 1", ip).Output()
// 	if err != nil {
// 		return "未找到"
// 	}
// 	if strings.Contains(string(out), "Received") && strings.Contains(string(out), "TTL") {
// 		return "在线"
// 	}
// 	if strings.Contains(string(out), "not find host") {
// 		return "离线"

// 	}
// 	return "未找到"
// }

// 获取 MAC 地址
func getMACAddress(ip string) (string, error) {
	// 执行 arp 命令
	var out []byte
	var err error
	// var filed int

	switch myos := runtime.GOOS; myos {
	case "windows":
		out, err = exec.Command("arp", "-a", ip).Output()
		// filed = 2
	case "plan9":
		out, err = exec.Command("arp", "-a", ip).Output()
		// filed = 2
	case "linux":
		out, err = exec.Command("arp", "-n", ip).Output()
		// filed = 3

	}

	if err != nil {
		return "未找到", fmt.Errorf("执行 arp 命令失败: %v", err)

	}

	// 解析 arp 命令的输出
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, ip) {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[1], nil
			}
		}
	}

	return "未找到", fmt.Errorf("未找到 IP 地址对应的 MAC 地址")
}

// 加载 OUI 数据库
func loadOUIDatabase(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ouiDB := make(map[string]string)
	reader := csv.NewReader(file)
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		if len(record) < 2 {
			continue
		}
		ouiDB[strings.ToUpper(record[1])] = record[2]
	}
	return ouiDB, nil
}

// 获取制造商信息
func getManufacturer(mac string) string {

	if len(mac) < 8 {
		return "未找到"
	}
	oui := strings.ReplaceAll(strings.ToUpper(mac[:8]), ":", "")
	oui = strings.ReplaceAll(oui, "-", "")
	if manufacturer, ok := ouiDB[oui]; ok {
		return manufacturer
	}
	return "未找到"
}
