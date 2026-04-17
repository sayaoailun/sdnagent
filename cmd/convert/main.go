package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Flow 表示流表规则
type Flow struct {
	Table    int
	Priority int
	Matches  string
	Actions  string
}

// 按照优先级从低到高排序
type ByPriority []Flow

func (a ByPriority) Len() int           { return len(a) }
func (a ByPriority) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPriority) Less(i, j int) bool { return a[i].Priority < a[j].Priority }

func main() {
	// 解析命令行参数
	inputFile := flag.String("input", "flow-ofctl.txt", "输入文件路径")
	flag.Parse()

	// 打开输入文件
	file, err := os.Open(*inputFile)
	if err != nil {
		fmt.Printf("无法打开文件: %v\n", err)
		return
	}
	defer file.Close()

	// 读取文件内容
	scanner := bufio.NewScanner(file)
	var flows []Flow

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// 解析行
		flow, err := parseFlow(line)
		if err != nil {
			fmt.Printf("解析错误: %v\n", err)
			continue
		}

		flows = append(flows, flow)
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("读取文件错误: %v\n", err)
		return
	}

	// 按照优先级排序
	sort.Sort(ByPriority(flows))

	// 输出结果
	for _, flow := range flows {
		fmt.Printf("table=%d, priority=%d", flow.Table, flow.Priority)
		if flow.Matches != "" {
			fmt.Printf(", %s", flow.Matches)
		}
		fmt.Printf(" actions=%s\n", flow.Actions)
	}
}

// parseFlow 解析一行流表规则
func parseFlow(line string) (Flow, error) {
	var flow Flow

	// 分离actions部分
	parts := strings.Split(line, " actions=")
	if len(parts) != 2 {
		return flow, fmt.Errorf("无效的行格式")
	}

	// 解析actions
	flow.Actions = parts[1]

	// 解析前面的部分
	// 先移除cookie部分
	frontPart := parts[0]
	// 处理可能的空格
	frontPart = strings.TrimSpace(frontPart)

	// 分割所有部分
	frontParts := strings.Split(frontPart, ",")
	var matches []string

	for _, part := range frontParts {
		// 去除空格
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "table=") {
			fmt.Sscanf(part, "table=%d", &flow.Table)
		} else if strings.HasPrefix(part, "priority=") {
			fmt.Sscanf(part, "priority=%d", &flow.Priority)
		} else if !strings.HasPrefix(part, "cookie=") &&
			!strings.HasPrefix(part, "duration=") &&
			!strings.HasPrefix(part, "n_packets=") &&
			!strings.HasPrefix(part, "n_bytes=") {
			// 提取matches部分
			if part != "" {
				matches = append(matches, part)
			}
		}
	}

	flow.Matches = strings.Join(matches, ",")

	return flow, nil
}
