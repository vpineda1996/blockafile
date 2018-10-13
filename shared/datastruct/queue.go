package datastruct

type Queue struct{
	enqueue, dequeue Stack
}

type QueueElement interface {}

func (q *Queue) Enqueue(n QueueElement){
	q.enqueue.Push(n)
}

func (q *Queue) Dequeue()(QueueElement, bool){
	v, ok := q.dequeue.Pop()
	if ok{
		return v, true
	}

	for {
		v, ok := q.enqueue.Pop()
		if !ok{
			break
		}

		q.dequeue.Push(v)
	}

	return q.dequeue.Pop()
}

func (q *Queue) IsInQueue(eq func(QueueElement) bool) bool {
	for _, v := range q.dequeue.v {
		if eq(v) {
			return true
		}
	}
	for _, v := range q.enqueue.v {
		if eq(v) {
			return true
		}
	}
	return false
}

type Stack struct{
	v []QueueElement
}

func (s *Stack)Push(n QueueElement){
	s.v=append(s.v, n)
}

func (s *Stack) Pop()(QueueElement, bool){
	if len(s.v) == 0 {
		return nil, false
	}

	lastIdx := len(s.v)-1
	v := s.v[lastIdx]
	s.v=s.v[:lastIdx]
	return v, true
}
