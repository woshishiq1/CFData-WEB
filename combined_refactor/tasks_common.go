package main

import (
    "context"
    "fmt"
    "sync"
)

func runBoundedWorkers(ctx context.Context, total, maxWorkers, progressEvery int, onProgress func(current, total int), work func(idx int)) bool {
    if total == 0 {
        return false
    }

    var wg sync.WaitGroup
    wg.Add(total)

    slots := make(chan struct{}, maxWorkers)
    var count int
    var countMutex sync.Mutex
    wasCanceled := false

    for i := 0; i < total; i++ {
        select {
        case <-ctx.Done():
            wasCanceled = true
            wg.Done()
            continue
        case slots <- struct{}{}:
        }

        go func(idx int) {
            defer func() {
                <-slots
                wg.Done()

                countMutex.Lock()
                count++
                current := count
                countMutex.Unlock()

                if onProgress != nil && (progressEvery <= 1 || current%progressEvery == 0 || current == total) {
                    onProgress(current, total)
                }
            }()

            select {
            case <-ctx.Done():
                return
            default:
            }

            work(idx)
        }(i)
    }

    wg.Wait()
    return wasCanceled
}

func reportNSBProgress(session *appSession, phase string, current, total int, text string) {
    percentage := 0.0
    if total > 0 {
        percentage = float64(current) / float64(total) * 100
    }
    session.sendWSMessage("nsb_progress", map[string]interface{}{
        "phase":   phase,
        "current": current,
        "total":   total,
        "percent": fmt.Sprintf("%.2f", percentage),
        "text":    text,
    })
}
