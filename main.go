package main

import (
	"flag"
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/process"
	"log"
	"os"
	"os/exec"
	"time"
)

const (
	DAEMON    = "daemon"
	FOREVER   = "forever"
	MAXCPU    = 0.13019 // CPU占用阈值
	MAXMEMORY = 20      //内存占用阈值
)

// DoSomething 业务进程，仅输出日志
func DoSomething() {
	fp, _ := os.OpenFile("./dosomething.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	log.SetOutput(fp)
	for {
		log.Printf("DoSomething running in PID: %d PPID: %d\n", os.Getpid(), os.Getppid())
		time.Sleep(time.Second * 5)
	}
}

// StripSlice 处理命令
func StripSlice(slice []string, element string) []string {
	for i := 0; i < len(slice); {
		if slice[i] == element && i != len(slice)-1 {
			slice = append(slice[:i], slice[i+1:]...)
		} else if slice[i] == element && i == len(slice)-1 {
			slice = slice[:i]
		} else {
			i++
		}
	}
	return slice
}

// SubProcess 开启子进程
func SubProcess(args []string) *exec.Cmd {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[-] Error: %s\n", err)
	}
	return cmd
}

// 监控业务进程
func monitor(pid int32) error {
	// 监控进程存在
	exist, err := process.PidExists(pid)

	if !exist {
		return fmt.Errorf("process dont exist")
	}
	if !exist || err != nil {
		return err
	}

	// 获取进程
	p, err := process.NewProcess(pid)
	if err != nil {
		return err
	}

	// 监控CPU
	counts, err := cpu.Counts(true)
	if err != nil {
		return err
	}
	cpuPercentIn3Second, err := p.Percent(time.Second * 3)
	if err != nil {
		return err
	}

	// 监控内存
	memInfo, err := p.MemoryInfo()
	mem := memInfo.RSS / 1024 / 1024

	fmt.Println("[*] "+getNowTime(), "service cpu:", cpuPercentIn3Second/float64(counts), "  service mem：", mem, "MB")

	if cpuPercentIn3Second/float64(counts) >= MAXCPU {
		return fmt.Errorf("cpuPercent is too high")
	}
	if mem > MAXMEMORY {
		return fmt.Errorf("memory is too high")
	}

	return nil
}

// 获取时间
func getNowTime() string {
	timeNow := time.Now().Format("2006-01-02 15:04:05")
	return timeNow
}
func main() {
	daemon := flag.Bool(DAEMON, false, "run in daemon")
	forever := flag.Bool(FOREVER, false, "run forever")
	flag.Parse()

	if *daemon {
		SubProcess(StripSlice(os.Args, "-"+DAEMON))
		fmt.Printf("[*] Daemon start running in PID: %d PPID: %d\n", os.Getpid(), os.Getppid())
		os.Exit(0)
	} else if *forever {

		fp, _ := os.OpenFile("./monitor.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		log.SetOutput(fp)

		cmd := SubProcess(StripSlice(os.Args, "-"+FOREVER))
		fmt.Printf("[*] Forever start running in PID: %d PPID: %d\n", os.Getpid(), os.Getppid())

		for {
			err := monitor(int32(cmd.Process.Pid))
			if err != nil {

				log.Printf("[*] monitor error: %s", err.Error())

				_ = cmd.Process.Kill()
				_ = cmd.Wait()

				cmd = SubProcess(StripSlice(os.Args, "-"+FOREVER))
				fmt.Printf("[*] Forever start running in PID: %d PPID: %d\n", os.Getpid(), os.Getppid())
			}
			time.Sleep(time.Second * 3)
		}

	} else {
		fmt.Printf("[*] Service start running in PID: %d PPID: %d\n", os.Getpid(), os.Getppid())
	}
	DoSomething()
}
