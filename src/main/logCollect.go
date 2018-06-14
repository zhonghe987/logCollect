package main

import (
    "fmt"
    "os/exec"
    "os"
    "os/signal"
    "strings"
    "time"
    "bytes"
    "runtime"
)

func cmdExec(cmd string) string {
     exec_cmd := exec.Command("sh", "-c", cmd)
     var stdout, stderr bytes.Buffer
     exec_cmd.Stdout = &stdout
     exec_cmd.Stderr = &stderr
     err := exec_cmd.Run()
     if err != nil {
	fmt.Printf("cmd.Run() failed with %s\n", err)
     }
     outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
     if len(errStr)> 0{
         fmt.Printf("err:\n%s\n", errStr)
     }
     return outStr
     
}

func memcli(log chan string){
     for {
     fmt.Printf("---%s",<-log)
     }
}


func pullLog(date time.Time, cmd string, log chan string){
    //hour := string(date.Hour())
    dateString := date.Format("2006-01-02")
    new_cmd :=cmd + dateString
    out := cmdExec(new_cmd)
    lens := len(out)
    ks := strings.Split(out[1:lens-4], "  ")
    for _,k := range(ks){ 
       log <- k
    }
}

func work(cmd string, log chan string){
    date := time.Now()
    minute := int(date.Minute())
    fmt.Println(minute) 
    if (minute < 20) {
        pullLog(date, cmd, log)
    }else{
        
        ticker := time.NewTicker(3 * time.Second)
        go func() {
        for t := range ticker.C {
            date := time.Now()
            fmt.Println("\n",t)
            pullLog(date, cmd, log)
        }
    }()
    }
}

func init() {
    runtime.GOMAXPROCS(runtime.NumCPU())
}

func main(){
     sigs := make(chan os.Signal, 1)  
     done := make(chan bool, 1)
     log := make(chan string, 10)
     signal.Notify(sigs, os.Interrupt, os.Kill)
     go func() {  
        sig := <-sigs  
        switch sig {  
        case os.Interrupt:  
            fmt.Println("signal: Interrupt")  
        case os.Kill:  
            fmt.Println("signal: Kill")  
        default:  
            fmt.Println("signal: Others")  
        }  
        done <- true  
    }()
    cmd := "sudo radosgw-admin log list --date="
    go work(cmd, log)
    go memcli(log)
    fmt.Println("awaiting signal") 
    <- done
    close(done)  
    fmt.Println("exiting")
}
