package main

import (
    "fmt"
    "os/exec"
    "os"
    "errors"
    "os/signal"
    "strings"
    "time"
    "syscall"
    "context"
    "bytes"
    "sync"
    "regexp"
    _ "strconv"
    "runtime"
    "github.com/olivere/elastic"
    "github.com/sdbaiguanghe/glog"
)
var (
    MaxWorker = 50
    MaxQueue  = 200000
    wg        sync.WaitGroup
)

type Job struct {
     object string
}
var client *elastic.Client
var JobQueue chan Job = make(chan Job, MaxQueue)

type Dispatcher struct {
}

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

func memcli(){ 
     var err error
     client, err = elastic.NewClient(elastic.SetURL("http://10.3.32.181:9200"),
                 elastic.SetSniff(false),
                 elastic.SetHealthcheckInterval(10*time.Second),
                 elastic.SetMaxRetries(5))
     if err != nil {
        fmt.Println(err)
        glog.Error("error glog")
     }
     exists, err := client.IndexExists("oss").Do(context.Background())
     if err != nil{
         fmt.Println(err)
         panic(err)
     }
     if !exists{
         _, err = client.CreateIndex("oss").Do(context.Background())
	     if err != nil {
		// Handle error
                fmt.Println(err)
		panic(err)
        }
    }
}

func entriRegexp(regs string, entri string) bool{
     entries := strings.Replace(entri, "\"", "", -1)
     reg, _ := regexp.Compile(regs)
     ok :=  reg.FindString(entries)
     if len(ok)>0{
         return true
    }
    return false
}


func doTask(log Job, id int) {
     object := log.object
     cmd := "sudo radosgw-admin log show --object="
     new_object := strings.Split(object, ",")[0]
     objects := strings.Replace(new_object, "\"", "", -1)  
     new_cmd := cmd + objects
     fmt.Printf("-+++++-%s\n", new_cmd)
     out, err := cmdExec(new_cmd)
     if err ==nil{
         _, err = client.Index().Index("oss").Type("log").Id(string(id)).BodyJson(out).Do(context.Background())
         fmt.Printf("00000ok\n")
         if err != nil {
             // Handle error
             fmt.Println(err)
             panic(err)
         }
         delete_log_cmd := "sudo radosgw-admin log rm --object="
         delete_log_cmd_new := delete_log_cmd + objects
         _, err = cmdExec(delete_log_cmd_new)
         if err != nil {
            // Handle error
            fmt.Println(err)
            panic(err)
         }
     }
}

func pullLog(date time.Time, cmd string) {
    out, _ := cmdExec(cmd)
    lens := len(out)
    if lens > 4{
       ks := strings.Split(out[1:lens-4], " ")[1:] 
       for _,k := range(ks){
            str := strings.Replace(k, " ", "", -1)  
            if len(str)> 14{
                if (entriRegexp(`^[\d]{4}-[\d]{2}-[\d]{2}-[\d]{2}`, k)){
                        fmt.Printf("---%s\n",k)
                        job := Job{object : k}
                        JobQueue <- job
                   } 
                }
            }
       }
}

type Worker struct {
    quit chan bool
}

func NewWorker() Worker {
    return Worker{
        quit: make(chan bool)}
}

func (w Worker) Start() {
    go func() {
        i := 0
        for {
            select {
            case job := <-JobQueue:
                // we have received a work request.

                doTask(job, i)
                i++ 
            case <-w.quit:
                // we have received a signal to stop
                return
            }
        }
    }()
}

func (w Worker) Stop() {
    go func() {
        w.quit <- true
    }()
}

func NewDispatcher() *Dispatcher {
    return &Dispatcher{}
}

func (d *Dispatcher) Run() {
    // starting n number of workers
    for i := 0; i < MaxWorker; i++ {
        worker := NewWorker()
        worker.Start()
    }
}


func init(){
    memcli()    
}

func main(){
    runtime.GOMAXPROCS(runtime.NumCPU())
    d := NewDispatcher()
    done := make(chan bool, 1) 
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt, os.Kill,  syscall.SIGUSR1, syscall.SIGUSR2)
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
    d.Run()
    cmd := "sudo radosgw-admin log list "
    ticker := time.NewTicker(30 * time.Second)
    go func() {
        for t := range ticker.C {
            date := time.Now()
            fmt.Println("\n",t)
            pullLog(date, cmd)
           }
    }()
    <- done
    close(done)  
    fmt.Println("exiting")
}