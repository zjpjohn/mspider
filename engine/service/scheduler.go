package service
import(
    "time"
    "github.com/zl-leaf/mspider/engine/msg"
    "github.com/zl-leaf/mspider/scheduler"
)

type SchedulerService struct {
    Scheduler *scheduler.Scheduler
    EventPublisher chan string
    Listener *SpiderService
    State int
    MessageHandler msg.ISchedulerMessageHandler
}

func (this *SchedulerService) Start() error {
    this.State = WorkingState
    go this.listen(this.Listener.EventPublisher)
    go this.push()
    return nil
}

func (this *SchedulerService) Stop() error {
    this.State = StopState
    return nil
}

func (this *SchedulerService) SetScheduler(s *scheduler.Scheduler) {
    this.Scheduler = s
}

func (this *SchedulerService) listen(listenerChan chan string) {
    for {
        if this.State == StopState {
            break
        }
        value := <- listenerChan
        this.do(value)
    }
}

func (this *SchedulerService) push() {
    for {
        u, err := this.Scheduler.Head()

        if err != nil {
            continue
        }

        u, err = this.MessageHandler.HandleResponse(u)
        if err != nil {
            continue
        }

        this.EventPublisher <- u
        time.Sleep(time.Duration(this.Scheduler.Interval))
    }
}

func (this *SchedulerService) do(content string) {
    u, err := this.MessageHandler.HandleRequest(content)
    if err == nil {
        this.Scheduler.Add(u)
    }
}

func CreateSchedulerService() (schedulerService *SchedulerService) {
    schedulerService = &SchedulerService{}
    schedulerService.EventPublisher = make(chan string)
    schedulerService.MessageHandler = &msg.SchedulerMessageHandler{}
    return
}