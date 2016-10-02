package service
import(
    "time"
    "github.com/zl-leaf/mspider/engine/msg"
    "github.com/zl-leaf/mspider/spider"
    "github.com/zl-leaf/mspider/spider/pool"
    "github.com/zl-leaf/mspider/logger"
)

const(
    stopSpiderWait = 1
    getSpiderRetryNum = 3
    getSpiderRetryWait = 1
)

type SpiderService struct {
    Pool *pool.SpiderPool
    EventPublisher chan string
    Listener *DownloaderService
    State int
    MessageHandler msg.ISpiderMessageHandler
    Validator msg.ISpiderValidator
}

func (this *SpiderService) Start() error {
    this.State = WorkingState
    go this.listen(this.Listener.EventPublisher)

    for _,s := range this.Pool.All() {
        for _, u := range s.StartURLs() {
            this.EventPublisher <- u
        }
    }
    return nil
}

func (this *SpiderService) Stop() error {
    this.State = StopState
    stopChan := make(chan string)
    go func(stopChan chan string) {
        for {
            allFree := true
            for _,free := range this.Pool.States() {
                if !free {
                    allFree = false
                    break
                }
            }
            if allFree {
                break
            }
            time.Sleep(time.Duration(stopSpiderWait) * time.Second)
        }
        stopChan <- "stop"
    }(stopChan)
    <- stopChan
    return nil
}

func (this *SpiderService) listen(listenerChan chan msg.SpiderRequest) {
    for {
        request := <- listenerChan
        if this.Validator != nil {
            if err := this.Validator.Validate(request); err != nil {
                logger.Error(logger.SYSTEM, err.Error())
                continue
            }
        }

        s, err := this.Pool.Get(request.URL)
        if err != nil {
            logger.Error(logger.SYSTEM, err.Error())
            return
        }
        go this.do(request, s)
    }
}

func (this *SpiderService) do(request msg.SpiderRequest, s *spider.Spider) {
    if this.State == StopState {
        return
    }

    err := s.Do(request.URL, request.Html)
    if err != nil {
        logger.Error(logger.SYSTEM, err.Error())
        return
    }
    defer this.Pool.Put(s)
    redirects := s.Redirects()
    logger.Info(logger.SYSTEM, "spider id: %s finish url: %s, got %d redirects", s.ID, request.URL, len(redirects))
    for _, u := range redirects {
        if this.State == StopState {
            break
        }
        this.EventPublisher <- u
    }
    return
}

func CreateSpiderService() (spiderService *SpiderService) {
    spiderService = &SpiderService{}
    spiderService.Pool = pool.New()
    spiderService.EventPublisher = make(chan string)
    return
}