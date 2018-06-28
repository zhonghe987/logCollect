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
)
var (
    MaxWorker = 50
    MaxQueue  = 2000
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
        panic("error glog")
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
func timeCom(time_1 time.Time, time_2 string) bool {
    tim_1 := time_1.Format("2006-01-02 15")
    time_2 = time_2[1:len(time_2)-1]
    t1, err := time.Parse("2006-01-02 15", tim_1)
    t2, err := time.Parse("2006-01-02 15", time_2)
    if err == nil && t2.Before(t1) {
       return true
    }    
    return false
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

func objectDelete(objects string){
     delete_log_cmd := "sudo radosgw-admin log rm --object=%s"
     delete_log_cmd_new := fmt.Sprintf(delete_log_cmd, objects)
     _, err := cmdExec(delete_log_cmd_new) 
     if err != nil {
        // Handle error
        panic(err)
     }
}
func doTask(log Job) {
     object := log.object
     cmd := "sudo radosgw-admin log show --object=%s"
     new_object := strings.Split(object, ",")[0]
     new_cmd := fmt.Sprintf(cmd, new_object)
     i := 0
     for{
         out, err := cmdExec(new_cmd)
         
         if err == nil{
             _, err = client.Index().Index("oss").Type("log").BodyJson(out).Do(context.Background())
             if err != nil {
                 panic(err)
             }
             objectDelete(new_object)
             break
         }
         panic(err)
         if i > 3{
            objectDelete(new_object)
            break
         }
         i++
     }
}

func pullLog(cmd string) {
    
    out, _ := cmdExec(cmd)
    lens := len(out)
    if lens > 4{
       date := time.Now()
       ks := strings.Split(out[1:lens-4], " ")[1:] 
       for _,k := range(ks){
            str := strings.Replace(k, " ", "", -1)  
            if len(str)> 14{
                log_date := str[:11]+" "+str[12:14]+"\""
                if (entriRegexp(`^[\d]{4}-[\d]{2}-[\d]{2}-[\d]{2}`, k) && timeCom(date, log_date)){
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
        for {
            select {
            case job := <-JobQueue:
                // we have received a work request.

                doTask(job)
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
    ticker := time.NewTicker(360 * time.Second)
    go func() {
        for t := range ticker.C {
            fmt.Println("\n",t)
            pullLog(cmd)
           }
    }()
    <- done
    close(done)  
    fmt.Println("exiting")
}
