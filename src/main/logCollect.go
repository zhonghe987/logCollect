package main

import (
    "fmt"
    "os/exec"
    "os"
    "os/signal"
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


func pullLog(cmd string){
     date := time.Now()
     minute := int(date.Minute())
     fmt.Println(minute) 
     if (minute > 20) {
         fmt.Printf("-------")
     }else{
       for {
         
         select{
            default:
                now_date := time.Now()
                hour := string(now_date.Hour())
                //dateString := now_date.Format("2006-01-02")
                new_cmd :=cmd + "2018-05-21"
                fmt.Printf(hour)
                out := cmdExec(new_cmd)
                ks := strings.Split(out, ",")
                for _, v :=  range(ks){
                  //if len(v)< 76{
                      fmt.Printf(v)
                  //}
                } 
         }
       }
    }
}


func main(){
     sigs := make(chan os.Signal, 1)  
     done := make(chan bool, 1)
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
    go pullLog(cmd)
    fmt.Println("awaiting signal") 
    <- done
    close(done)  
    fmt.Println("exiting")
}
