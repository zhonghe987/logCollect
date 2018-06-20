package main

import (
    "fmt"
    "os/exec"
    "os"
    "errors"
    "os/signal"
    "strings"
    "time"
    "context"
    "bytes"
    _ "strconv"
    "runtime"
    "github.com/olivere/elastic"
    "github.com/sdbaiguanghe/glog"
)

func cmdExec(cmd string) (string, error){
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
         return "error", errors.New("errStr")
     }
     return outStr, nil
     
}

func memcli(log chan string){
     client, err := elastic.NewClient(elastic.SetURL("http://10.3.32.181:9200"),
                 elastic.SetSniff(false),
                 elastic.SetHealthcheckInterval(10*time.Second),
                 elastic.SetMaxRetries(5))
     if err != nil {
        glog.Error("error glog")
     }
     exists, err := client.IndexExists("oss").Do(context.Background())
     if err != nil{
         panic(err)
     }
     if !exists{
         _, err = client.CreateIndex("oss").Do(context.Background())
	     if err != nil {
		// Handle error
		panic(err)
        }
    }
     i :=1
     for {
         select{
          case object := <- log:
              cmd := "sudo radosgw-admin log show --object="
              new_object := strings.Split(object, ",")[0]
              objects := strings.Replace(new_object, "\"", "", -1)  
              new_cmd := cmd + objects
              fmt.Printf("---%s\n", new_cmd)
              out, err := cmdExec(new_cmd)
              if err !=nil{
                  continue
              }
              _, err = client.Index().Index("oss").Type("log").Id(string(i)).BodyJson(out).Do(context.Background())
              fmt.Printf("00000ok\n")
              if err != nil {
                  // Handle error
                  panic(err)
              }
              delete_log_cmd := "sudo radosgw-admin log rm --object="
              delete_log_cmd_new := delete_log_cmd + objects
              _, err = cmdExec(delete_log_cmd_new)
              if err != nil {
                  // Handle error
                  panic(err)
              }
              i++
         default:
              time.Sleep(10 * time.Second)
         }
     }
}


func pullLog(date time.Time, cmd string, log chan string){
    //hour := string(date.Hour())
    dateString := date.Format("2006-01-02")
    new_cmd :=cmd + dateString
    out, _ := cmdExec(new_cmd)
    lens := len(out)
    fmt.Println(lens)
    if lens > 4{
       ks := strings.Split(out[1:lens-4], " ")[1:] 
       for _,k := range(ks){
            fmt.Printf(k)
            log <- k
       }
    }
}

func work(cmd string, log chan string){
    date := time.Now()
    minute := int(date.Minute())
    fmt.Println(minute) 
    if (minute <  20) {
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
