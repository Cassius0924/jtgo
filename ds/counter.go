package ds

type Counter struct {
	value        int64
	defaultValue int64
}

// NewCounter 创建计数器
func NewCounter(defaultValue int64) *Counter {
	return &Counter{
		value:        defaultValue,
		defaultValue: defaultValue,
	}
}

// Inc 自增计数器
func (c *Counter) Inc() {
	c.value++
}

// Dec 自减计数器
func (c *Counter) Dec() {
	c.value--
}

// Reset 重置计数器
func (c *Counter) Reset() {
	c.value = c.defaultValue
}

// Set 设置计数器值
func (c *Counter) Set(value int64) {
	c.value = value
}

// Value 获取计数器值
func (c *Counter) Value() int64 {
	return c.value
}
