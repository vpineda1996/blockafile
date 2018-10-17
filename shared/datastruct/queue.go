package datastruct

type Queue struct{
	enqueue, dequeue Stack
}

type QueueElement interface {}

func (queue *Queue) Enqueue(n QueueElement){
	queue.enqueue.Push(n)
}

func (queue *Queue) Dequeue()(QueueElement, bool){
	v, ok := queue.dequeue.Pop()
	if ok{
		return v, true
	}

	for {
		v, ok := queue.enqueue.Pop()
		if !ok{
			break
		}

		queue.dequeue.Push(v)
	}

	return queue.dequeue.Pop()
}

func (queue *Queue) IsInQueue(eq func(QueueElement) bool) bool {
	for _, v := range queue.dequeue.arr {
		if eq(v) {
			return true
		}
	}
	for _, v := range queue.enqueue.arr {
		if eq(v) {
			return true
		}
	}
	return false
}

func (queue *Queue) Len() int {
	return len(queue.enqueue.arr) + len(queue.dequeue.arr)
}

type Stack struct{
	arr []QueueElement
}

func (s *Stack)Push(n QueueElement){
	s.arr = append(s.arr, n)
}

func (s *Stack) Pop()(QueueElement, bool){
	if len(s.arr) == 0 {
		return nil, false
	}

	lastIdx := len(s.arr)-1
	v := s.arr[lastIdx]
	s.arr =s.arr[:lastIdx]

	return v, true
}
