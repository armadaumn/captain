package captain

import "sync"

const LIMIT = 10

type resQueue struct {
	lock    sync.Mutex
	average float64
	log     []HistoryLog
}

type ResQueue interface {
	Push(item HistoryLog)
	Pop() *HistoryLog
	Peek() *HistoryLog
	Average() float64
	Len() int
	GetRecentUpdate() map[string]float64
}

type HistoryLog struct {
	containers map[string]float64
	sum        float64
}

func initQueue() ResQueue {
	q := &resQueue{
		average: 0.0,
		log: make([]HistoryLog, 0, LIMIT),
	}
	return q
}

func (q *resQueue) Push(item HistoryLog) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if len(q.log) == LIMIT {
		q.popAveUpdate()
		q.log = q.log[1:]
	}
	q.pushAveUpdate(item.sum)
	q.log = append(q.log, item)
}

func (q *resQueue) Pop() *HistoryLog {
	q.lock.Lock()
	defer q.lock.Unlock()

	if len(q.log) == 0 {
		return nil
	}

	item := q.log[0]
	q.log = q.log[1:]
	return &item
}

func (q *resQueue) Peek() *HistoryLog {
	q.lock.Lock()
	defer q.lock.Unlock()

	if len(q.log) == 0 {
		return nil
	}

	item := q.log[0]
	return &item
}

func (q *resQueue) Average() float64 {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.average
}

func (q *resQueue) Len() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return len(q.log)
}

func (q *resQueue) GetRecentUpdate() map[string]float64 {
	q.lock.Lock()
	defer q.lock.Unlock()
	lastIndex := len(q.log) - 1
	if lastIndex < 0 {
		return nil
	}
	return q.log[lastIndex].containers
}

func (q *resQueue) popAveUpdate() {
	total := q.average * float64(len(q.log))
	total -= q.log[0].sum
	q.average = total / float64(len(q.log) - 1)
}

func (q *resQueue) pushAveUpdate(sum float64) {
	total := q.average * float64(len(q.log))
	total += sum
	q.average = total / float64(len(q.log) + 1)
}
