package main

import (
    "fmt"
    "os/exec"
    _ "os"
    "strings"
    "time"
    "bytes"
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


func pullLog(date chan int, cmd string , sig chan int){
    for {
         select{
            case <- date:
                now_date := time.Now()
                hour := string(now_date.Hour())
                dateString := now_date.Format("2006-01-02")
                new_cmd :=cmd + dateString
                fmt.Printf(hour)
                out := cmdExec(new_cmd)
                ks := strings.Split(out, ",")
                for _, v :=  range(ks){
                  if len(v)< 76{
                      fmt.Printf("----"+v)
                  }
                } 
                sig <- 1
            default:
                fmt.Println("exit")
         }
    }
}


func main(){
     date := make(chan int, 1)
     sig := make(chan int)
     cmd := "sudo radosgw-admin log list --date="
     go pullLog(date, cmd, sig)
     //date_now := time.Now()
     date <- 1
     <-sig
     defer close(date)
     defer close(sig)
     
}
