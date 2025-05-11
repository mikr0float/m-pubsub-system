package subpub

import (
	"context"
	"errors"
	"slices"
	"sync"
)

// MessageHandler это колл-бек ф-ция, обрабатывающая сообщения, доставленные к подписчикам
type MessageHandler func(msg interface{})

// subPub реализация интерфейса шины событий
type SubPub struct {
	subscribers map[string][]chan interface{}
	mu          sync.RWMutex
	wg          sync.WaitGroup
	closed      bool
}

// ErrBusClosed ошибка доступа к Subscription
var ErrBusClosed error = errors.New("subPub: bus is closed")

// NewSubPub создает шину событий публикующего
func NewSubPub() *SubPub {
	return &SubPub{
		subscribers: make(map[string][]chan interface{}),
	}
}

// Subscription реализация подписки
type Subscription struct {
	subject string
	ch      chan interface{}
	bus     *SubPub
}

// Subscribe подписывает подписчика на публикующего subject
func (s *SubPub) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return Subscription{}, ErrBusClosed
	}

	ch := make(chan interface{}, 100)
	// Слайс каналов подписки
	s.subscribers[subject] = append(s.subscribers[subject], ch)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for message := range ch {
			cb(message)
		}
	}()
	// ???????????????????????????????????????????????????????????????????????
	// Нужно ли здесь поставить wg.Wait()????
	return Subscription{
		subject: subject,
		ch:      ch,
		bus:     s,
	}, nil
}

// Unsubscribe удаляет подписку
func (s *Subscription) Unsubscribe() {
	s.bus.unsubscribe(s.subject, s.ch)
}

// unsubscribe удаляет канал из подписок и завершает обработку сообщений обработчиком
func (s *SubPub) unsubscribe(subject string, ch chan interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subs, ok := s.subscribers[subject]
	if !ok {
		return
	}

	for i, sub := range subs {
		if sub == ch {
			s.subscribers[subject] = slices.Delete(subs, i, i+1)
			close(ch)
			break
		}
	}
}

// Publish публикует сообщение в каналы подписчиков subject
func (s *SubPub) Publish(subject string, msg interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return ErrBusClosed
	}

	for _, ch := range s.subscribers[subject] {
		ch <- msg
	}
	return nil
}

// Close завершает работу шины subPub
func (s *SubPub) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrBusClosed
	}
	s.closed = true
	s.mu.Unlock()

	s.mu.Lock()
	for subject, subs := range s.subscribers {
		for _, ch := range subs {
			close(ch)
		}
		delete(s.subscribers, subject)
	}
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
