// Package cache предоставляет реализацию LRU-кэша с поддержкой TTL
package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

/*
ILRUCache описывает интерфейс LRU-кэша. Он поддерживает только строковые ключи и простые типы данных в значениях.

Все методы интерфейса являются потокобезопасными.
*/
type ILRUCache interface {
	// Put добавляет или обновляет запись в кэше с заданным TTL.
	// Если TTL <= 0, используется значение c.defaultTTL.
	// При переполнении кэша (количество элементов >= capacity) удаляется LRU-элемент.
	Put(ctx context.Context, key string, value interface{}, ttl time.Duration) error

    // Get возвращает данные из кэша по ключу.
	// Если данные не найдены или их TTL истёк, возвращается ErrKeyNotFound.
	Get(ctx context.Context, key string) (value interface{}, expiresAt time.Time, err error)

    // GetAll получение всего наполнения кэша в виде двух слайсов: слайса ключей и слайса значений.
	// Пары ключ-значения из кэша располагаются на соответствующих позициях в слайсах.
	GetAll(ctx context.Context) (keys []string, values []interface{}, err error)
   
	// Evict ручное удаление данных по ключу
	// Если ключ не найден — возвращает ErrKeyNotFound.
    Evict(ctx context.Context, key string) (value interface{}, err error)

    // EvictAll ручная инвалидация всего кэша
	EvictAll(ctx context.Context) error 
}

// ErrKeyNotFound сигнализирует о том, что ключ не существует.
var (
	ErrKeyNotFound = errors.New("key not found")
)

// item хранит данные записи кэша.
type item struct {
	key       string
	value     interface{}
	expiresAt time.Time
}

// ListNode представляет узел двусвязного списка, используемого
// для управления порядком "least recently used".
type ListNode struct{
	data *item
	prev *ListNode
	next *ListNode
}

// LRUCache реализует интерфейс ILRUCache,
// используя двусвязный список + map для O(1)-доступа к элементам.
type LRUCache struct {
	mu			sync.RWMutex
    capacity 	int
    cache 		map[string]*ListNode
	defaultTTL  time.Duration
    left 		*ListNode // Least Recently Used
    right 		*ListNode // Most Recently Used
}

// NewLRUCache создаёт новый LRUCache с заданной ёмкостью (capacity)
// и временем жизни по умолчанию (defaultTTL).
func NewLRUCache(capacity int, defaultTTL time.Duration) ILRUCache {
    return &LRUCache{
        capacity: capacity,
        cache: make(map[string]*ListNode, capacity),
		defaultTTL: defaultTTL,
	}
}

// Put добавляет или обновляет запись в кэше с указанным TTL.
func (c *LRUCache) Put(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	expiresAt := time.Now().Add(ttl)

	if node, ok := c.cache[key]; ok {
		node.data.value = value
		node.data.expiresAt = expiresAt
		c.moveToFront(node)
		return nil
	}

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

// Get возвращает значение и время истечения TTL для заданного ключа.
func (c *LRUCache) Get(ctx context.Context, key string) (interface{}, time.Time, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.cache[key]
	if !ok {
		return nil, time.Time{}, ErrKeyNotFound
	}

	if time.Now().After(node.data.expiresAt) {
		c.removeNode(node)
		delete(c.cache, key)
		return nil, time.Time{}, ErrKeyNotFound
	}

	c.moveToFront(node)

	return node.data.value, node.data.expiresAt, nil
}

// GetAll Получение всего текущего наполнения кэша в виде двух списков: списка ключей и списка значений.
// Пары ключ-значение располагаются на соответствующих индексах.
func (c *LRUCache) GetAll(ctx context.Context) ([]string, []interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.cache) == 0 {
		return nil, nil, nil
	}

	keys := make([]string, 0, len(c.cache))
	values := make([]interface{}, 0, len(c.cache))

	current := c.right
	for current != nil {

		if time.Now().Before(current.data.expiresAt) {
			keys = append(keys, current.data.key)
			values = append(values, current.data.value)
		}
		current = current.next
	}

	return keys, values, nil
}

// Evict удаляет элемент по ключу из кэша.
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

// EvictAll полностью очищает кэш.
func (c *LRUCache) EvictAll(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.right = nil
	c.left = nil
	c.cache = make(map[string]*ListNode, c.capacity)
	return nil
} 



// moveToFront перемещает заданный узел в начало очереди (right).
func (c *LRUCache) moveToFront(node *ListNode){
	
	if node == c.right {
		return
	}
	c.removeNode(node)
	c.addToFront(node)
}

// removeNode удаляет узел из двусвязного списка.
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

// addToFront добавляет узел в начало списка (right).
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

// removeLeastUsed удаляет наиболее "старый" элемент (left) из списка и map.
func (c *LRUCache) removeLeastUsed(){
	if c.left == nil {
		return
	}
	oldLeft := c.left
	c.removeNode(oldLeft)
	delete(c.cache, oldLeft.data.key)
}


