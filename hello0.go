// package main

// import (
// 	"encoding/csv"
// 	"fmt"
// 	"net"
// 	"os"
// 	"runtime"
// 	"strings"
// 	"sync"
// 	"time"

// 	"github.com/go-ping/ping"
// )

// var wg sync.WaitGroup
// var sem chan struct{}

// func main() {
// 	runtime.GOMAXPROCS(runtime.NumCPU())
// 	sem = make(chan struct{}, 5000)

// 	fmt.Println("状态\t\t名称\t\t\t\tIP\t\t制造商\t\tMAC地址\n")

// 	_, err := getLocalIPs()
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return
// 	}

// 	wg.Wait()
// }

// // type networkDetail  struct{
// // 	status string
// // 	name string
// // 	ip string
// // 	manufacturer string
// // 	mac string
// // }

// // 获取当前局域网内的所有 IP 地址及其状态和制造商信息
// func getLocalIPs() ([]map[string]string, error) {
// 	var details []map[string]string

// 	// 	type Interface struct {
// 	//     Index        int          // 索引，>=1的整数
// 	//     MTU          int          // 最大传输单元
// 	//     Name         string       // 接口名，例如"en0"、"lo0"、"eth0.100"
// 	//     HardwareAddr HardwareAddr // 硬件地址，IEEE MAC-48、EUI-48或EUI-64格式
// 	//     Flags        Flags        // 接口的属性，例如FlagUp、FlagLoopback、FlagMulticast
// 	// }
// 	ifaces, err := net.Interfaces() // interface 表示网络接口，
// 	if err != nil {
// 		fmt.Println("获取网络接口失败 err=", err)
// 		return nil, err
// 	}

// 	ouiDB, err := loadOUIDatabase("oui.csv") // 加载 OUI 数据库
// 	if err != nil {
// 		fmt.Println("加载 OUI 数据库失败 err=", err)
// 		return nil, err
// 	}

// 	for _, iface := range ifaces { // iface: 网络接口

// 		// 	type Addr interface {
// 		//     Network() string // 网络名
// 		//     String() string  // 字符串格式的地址
// 		// }
// 		// 获取网络接口中的地址
// 		addrs, err := iface.Addrs() // 返回一个或多个接口地址 addr:接口地址
// 		if err != nil {
// 			fmt.Println(err)
// 			continue
// 		}

// 		for _, addr := range addrs {

// 			// type IPNet struct {
// 			//     IP   IP     // 网络地址
// 			//     Mask IPMask // 子网掩码
// 			// }
// 			ipnet, ok := addr.(*net.IPNet)
// 			if ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
// 				CIDR := ipnet.String()
// 				fmt.Printf("当前搜索CIDR: %v请耐心等待.......\n", CIDR)
// 				// fmt.Println("状态\t\t名称\t\t\t\tIP\t\t制造商\t\tMAC地址\n")

// 				scanAndPrintCIDR(CIDR, &iface, ouiDB)

// 				// detail := make(map[string]string)

// 				// detail["name"] = iface.Name // 接口名称

// 				// ip := ipnet.IP.String() // ip地址
// 				// detail["ip"] = ip

// 				// status := getIPStatus(ip) //ip地址状态
// 				// detail["status"] = status

// 				// mac := iface.HardwareAddr.String() // mac 地址
// 				// detail["mac"] = mac

// 				// manufacturer := getManufacturer(mac, ouiDB) // 根据 mac 获取制造商信息
// 				// detail["manufacturer"] = manufacturer

// 				// // 添加数据
// 				// details = append(details, detail)

// 				// fmt.Printf("%s\t\t %s\t\t\t\t %s\t\t %s\t\t%s\n", detail["status"], detail["name"], detail["ip"], detail["manufacturer"], detail["mac"])

// 			}
// 		}
// 	}

// 	return details, nil
// }

// // 扫描并打印 CIDR 中所有的 ip 信息
// func scanAndPrintCIDR(CIDR string, iface *net.Interface, ouiDB map[string]string) {
// 	// 解析 CIDR
// 	ip, ipnet, err := net.ParseCIDR(CIDR) // ip: IP类型是代表单个IP地址的[]byte切片。本包的函数都可以接受4字节（IPv4）和16字节（IPv6）的切片作为输入。
// 	if err != nil {
// 		fmt.Println("解析 CIDR 失败 err=", err)
// 		return
// 	}
// 	ipAddr := net.IP(ip)
// 	mask := ipnet.Mask

// 	network := ipAddr.Mask(mask)
// 	broadcast := net.IP(make([]byte, 4))
// 	for i := 0; i < 4; i++ {
// 		broadcast[i] = network[i] | ^mask[i]
// 	}

// 	// 遍历 CIDR 中的所有 IP 地址
// 	for ip := ip.Mask(ipnet.Mask); !ip.Equal(broadcast); inc(ip) {
// 		sem <- struct{}{}
// 		wg.Add(1)
// 		go func(ip string) {
// 			defer wg.Done()
// 			defer func() {
// 				<-sem
// 			}()
// 			status := getIPStatus(ip)
// 			// fmt.Printf("ip: %v\n", ip)
// 			if status == "online" {
// 				// fmt.Printf("ip: %v\n", ip)

// 				detail := make(map[string]string)

// 				detail["name"] = iface.Name // 接口名称

// 				mac := iface.HardwareAddr.String() // mac 地址
// 				detail["mac"] = mac

// 				manufacturer := getManufacturer(mac, ouiDB) // 根据 mac 获取制造商信息
// 				detail["manufacturer"] = manufacturer

// 				// 添加数据
// 				// details = append(details, detail)

// 				// fmt.Printf("%s\t\t %s\t\t\t\t %s\t\t %s\t\t%s\n", status, iface.Name, ip, manufacturer, mac)
// 				fmt.Println(ip)
// 			}

// 		}(ip.String())

// 	}

// }

// // IP 地址递增
// func inc(ip net.IP) { //192.168.0.1

// 	for j := len(ip) - 1; j >= 0; j-- {
// 		ip[j]++
// 		if ip[j] > 0 {
// 			break
// 		}
// 	}
// }

// // 使用 ping 库获取 IP 地址的状态
// func getIPStatus(ip string) string {
// 	pinger, err := ping.NewPinger(ip)
// 	if err != nil {
// 		return "offline"
// 	}
// 	pinger.Count = 1
// 	pinger.Timeout = time.Second * 1
// 	pinger.SetPrivileged(true)

// 	err = pinger.Run() // Blocks until finished.
// 	if err != nil {
// 		return "offline"
// 	}
// 	stats := pinger.Statistics()
// 	if stats.PacketsRecv > 0 {
// 		return "online"
// 	}
// 	return "offline"
// }

// // 加载 OUI 数据库
// func loadOUIDatabase(filename string) (map[string]string, error) {
// 	file, err := os.Open(filename)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()

// 	ouiDB := make(map[string]string)
// 	reader := csv.NewReader(file)
// 	for {
// 		record, err := reader.Read()
// 		if err != nil {
// 			break
// 		}
// 		if len(record) < 2 {
// 			continue
// 		}
// 		ouiDB[strings.ToUpper(record[1])] = record[2]
// 	}

// 	return ouiDB, nil
// }

// // 获取制造商信息
// func getManufacturer(mac string, ouiDB map[string]string) string {
// 	if len(mac) < 8 {
// 		return "Unknown"
// 	}
// 	oui := strings.ReplaceAll(strings.ToUpper(mac[:8]), ":", "")

// 	if manufacturer, ok := ouiDB[oui]; ok {
// 		return manufacturer
// 	}
// 	return "Unknown"
// }
