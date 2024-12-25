package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

/*Идея для доработки реализовать фоновую очистку кэша с просроченными TTL
минусы - при большом объёме кэша из за того, что придётся лочить приведёт к потере производительности
Либо же добавить комбинацию мин-куча + «очистка при доступе»*/

// ILRUCache интерфейс LRU-кэша. Поддерживает только строковые ключи. Поддерживает только простые типы данных в значениях.
type ILRUCache interface {
	// Put запись данных в кэш
	Put(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    // Get получение данных из кэша по ключу
	Get(ctx context.Context, key string) (value interface{}, expiresAt time.Time, err error)
    // GetAll получение всего наполнения кэша в виде двух слайсов: слайса ключей и слайса значений. Пары ключ-значения из кэша располагаются на соответствующих позициях в слайсах.
	GetAll(ctx context.Context) (keys []string, values []interface{}, err error)
    // Evict ручное удаление данных по ключу
    Evict(ctx context.Context, key string) (value interface{}, err error)
    // EvictAll ручная инвалидация всего кэша
	EvictAll(ctx context.Context) error
}

var (
	ErrKeyNotFound = errors.New("key not found")
)

//в item будет храниться значения элемента кэша
type item struct {
	key       string
	value     interface{}
	expiresAt time.Time
}

//двусвязный список для реализации удаления из середины очереди за О(1)
type ListNode struct{
	data *item
	prev *ListNode
	next *ListNode
}

//реализация кэша
type LRUCache struct {
	mu			sync.RWMutex
    capacity 	int
    cache 		map[string]*ListNode
	defaultTTL  time.Duration
    left 		*ListNode // Least Recently Used
    right 		*ListNode // Most Recently Used
}

func NewLRUCache(capacity int, defaultTTL time.Duration) ILRUCache {
    return &LRUCache{
        capacity: capacity,
        cache: make(map[string]*ListNode, capacity),
		defaultTTL: defaultTTL,
	}
}

func (c *LRUCache) Put(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	expiresAt := time.Now().Add(ttl)
	//обновление, если уже есть в кэше
	if node, ok := c.cache[key]; ok {
		node.data.value = value
		node.data.expiresAt = expiresAt
		//перемещаем в начало очереди
		c.moveToFront(node)
		return nil
	}

	// Если размер достиг capacity — удаляем наименее используемый элемент
	if len(c.cache) >= c.capacity {
		c.removeLeastUsed()
	}

	newNode := &ListNode{
		data: &item{
			key:       key,
			value:     value,
			expiresAt: expiresAt,
		},
	}
	c.cache[key] = newNode
	c.addToFront(newNode)

	return nil
}

func (c *LRUCache) Get(ctx context.Context, key string) (interface{}, time.Time, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.cache[key]
	if !ok {
		return nil, time.Time{}, ErrKeyNotFound
	}

	//проверяем, не истёк ли TTL
	if time.Now().After(node.data.expiresAt) {
		//удаляем просроченный
		c.removeNode(node)
		delete(c.cache, key)
		return nil, time.Time{}, ErrKeyNotFound
	}

	//переместить в начало очереди
	c.moveToFront(node)

	return node.data.value, node.data.expiresAt, nil
}

func (c *LRUCache) GetAll(ctx context.Context) ([]string, []interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.cache) == 0 {
		return nil, nil, nil
	}

	keys := make([]string, 0, len(c.cache))
	values := make([]interface{}, 0, len(c.cache))

	//проход по списку для вывода от начала очереди до конца
	current := c.right
	for current != nil {
		// Проверяем TTL "на лету" — если истёк, пропускаем (но удалять тут не будем,
		// чтобы не усложнять логику и не блокировать на запись)
		if time.Now().Before(current.data.expiresAt) {
			keys = append(keys, current.data.key)
			values = append(values, current.data.value)
		}
		current = current.next
	}

	return keys, values, nil
}

func (c *LRUCache) Evict(ctx context.Context, key string) (interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.cache[key]
	if !ok {
		return nil, ErrKeyNotFound
	}

	val := node.data.value
	c.removeNode(node)
	delete(c.cache, key)

	return val, nil
}

func (c *LRUCache) EvictAll(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.right = nil
	c.left = nil
	c.cache = make(map[string]*ListNode, c.capacity)
	return nil
} 



//функция для перемещения в начало очереди
func (c *LRUCache) moveToFront(node *ListNode){
	//если и так в начале очереди
	if node == c.right {
		return
	}
	c.removeNode(node)
	c.addToFront(node)
}

// 		конец
// 		очереди				 						начало
// 		LRU элемент           						очереди
//(left) 1 <-> 2 <-> (next<-) 3 (->prev) <-> 4 <-> 5(right)

func (c *LRUCache) removeNode(node *ListNode) {
	if node.prev != nil { //проверка под вопросом
		node.prev.next = node.next
	} else {
		c.right = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		c.left = node.prev
	}
}

func (c *LRUCache) addToFront(node *ListNode){
	node.prev = nil
	node.next = c.right

	if c.right != nil{
		c.right.prev = node
	}
	c.right = node

	if c.left == nil {
		c.left = node
	}
}

func (c *LRUCache) removeLeastUsed(){
	if c.left == nil {
		return
	}
	oldLeft := c.left
	c.removeNode(oldLeft)
	delete(c.cache, oldLeft.data.key)
}


