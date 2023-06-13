package lru

import "container/list"

// Cache LRU cache. 并发不安全，包括一个 map 和一个双向链表
type Cache struct {
	maxBytes int64
	nbytes   int64
	ll       *list.List
	cache    map[string]*list.Element
	// optional and executed when an entry is purged.
	OnEvicted func(key string, value Value)
}

// 数据基础单元
type entry struct {
	key   string
	value Value
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

// New 新建一个 Cache 结构
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,                       // 限制大小
		ll:        list.New(),                     // 实现层为双向链表
		cache:     make(map[string]*list.Element), // 用户层为字典
		OnEvicted: onEvicted,                      // 这里传一个析构函数进来，进行销毁
	}
}

// Add 向缓存中添加元素，从这里可以看出 map 和队列是相互独立的
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele) // 如果有这个元素直接挪到最前面(左边)，完成 lru 的步骤
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value // 更新值
	} else {
		ele := c.ll.PushFront(&entry{key, value})        //将数据塞入队列头(左边)
		c.cache[key] = ele                               // 设置键值对
		c.nbytes += int64(len(key)) + int64(value.Len()) // 计算消耗比特长度
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes { // 如果大小超了就要弹出老数据
		c.RemoveOldest()
	}
}

// Get 通过 key 获取值
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry) // 有数据的话获得值并返回
		return kv.value, true
	}
	return // ? 没有数据的话直接返回，这里两个 nil 可以直接返回？
}

// RemoveOldest 移除 lru 数据
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() // 取最后一个元素
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key) // 删除字典中值
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value) // 如果销毁时的函数不为空，则执行该函数
		}
	}
}

// Len 直接返回缓存长度
func (c *Cache) Len() int {
	return c.ll.Len()
}
