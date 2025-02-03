package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

func main() {
	// 添加命令行参数解析
	port := flag.String("port", "8156", "HTTP server port")
	bind := flag.String("bind", "0.0.0.0", "HTTP server bind address")
	flag.Parse()

	// 设置 HTTP 处理函数
	http.HandleFunc("/", actionHandler)

	// 启动 HTTP 服务器
	addr := fmt.Sprintf("%s:%s", *bind, *port)
	log.Printf("Starting server at %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}

// 添加 HTTP 处理函数
func actionHandler(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")
	if action == "" {
		http.Error(w, "Action parameter is required", http.StatusBadRequest)
		return
	}

	delayStr := r.URL.Query().Get("delay")

	var delay = 1

	if delayStr == "" {
		delay = 1
	} else {
		var err error
		delay, err = strconv.Atoi(delayStr)
		if err != nil {
			delay = 1
		}
	}

	go func() {
		err := func() error {
			// 延迟执行
			time.Sleep(time.Duration(delay) * time.Second)

			switch action {
			case "shutdown":
				return exec.Command("shutdown", "/s", "/t", "0").Run()
			case "sleep":
				// 使用原生 Windows API 调用
				return triggerSleep()
			case "hibernate":
				return exec.Command("rundll32.exe", "powrprof.dll,SetSuspendState", "hibernate").Run()
			case "logout":
				return exec.Command("shutdown", "/l").Run()
			case "lock":
				return exec.Command("rundll32.exe", "user32.dll,LockWorkStation").Run()
			default:
				http.Error(w, "Invalid action parameter", http.StatusBadRequest)
				return nil
			}
		}()
		if err != nil {
			fmt.Printf("Error performing action: %s", err)
		}
	}()

	w.WriteHeader(http.StatusOK)
	fprintf, err := fmt.Fprintf(w, "Action '%s' will be performed after %d seconds", action, delay)
	if err != nil {
		fmt.Printf("Error writing response: %d", fprintf)
		return
	}
}

func triggerSleep() error {
	dll := syscall.NewLazyDLL("Powrprof.dll")
	proc := dll.NewProc("SetSuspendState")

	// 参数说明：
	// 参数1: Hibernate (0=睡眠, 1=休眠)
	// 参数2: ForceCritical (0=正常挂起)
	// 参数3: DisableWakeEvent (0=允许唤醒事件)
	ret, _, err := proc.Call(0, 0, 0)

	// API 返回非零表示成功
	if ret != 0 {
		return nil
	}

	// 错误处理
	if err != nil && err.Error() != "The operation completed successfully." {
		return fmt.Errorf("API 调用失败 (0x%X): %v", uintptr(ret), err)
	}
	return nil
}
