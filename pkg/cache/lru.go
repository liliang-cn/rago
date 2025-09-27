package cache

// lruNode represents a node in the LRU doubly linked list
type lruNode struct {
	key  string
	prev *lruNode
	next *lruNode
}

// lruList implements a doubly linked list for LRU cache
type lruList struct {
	head *lruNode
	tail *lruNode
	nodes map[string]*lruNode
}

// newLRUList creates a new LRU list
func newLRUList() *lruList {
	head := &lruNode{}
	tail := &lruNode{}
	head.next = tail
	tail.prev = head
	
	return &lruList{
		head:  head,
		tail:  tail,
		nodes: make(map[string]*lruNode),
	}
}

// addToFront adds a key to the front of the list
func (l *lruList) addToFront(key string) {
	// Remove existing node if present
	if node, exists := l.nodes[key]; exists {
		l.removeNode(node)
	}
	
	// Create new node
	node := &lruNode{key: key}
	
	// Insert at front (after head)
	node.prev = l.head
	node.next = l.head.next
	l.head.next.prev = node
	l.head.next = node
	
	// Store in map
	l.nodes[key] = node
}

// moveToFront moves an existing key to the front
func (l *lruList) moveToFront(key string) {
	if node, exists := l.nodes[key]; exists {
		l.removeNode(node)
		l.addToFront(key)
	}
}

// remove removes a key from the list
func (l *lruList) remove(key string) {
	if node, exists := l.nodes[key]; exists {
		l.removeNode(node)
		delete(l.nodes, key)
	}
}

// removeLast removes and returns the least recently used key
func (l *lruList) removeLast() string {
	if l.tail.prev == l.head {
		return "" // List is empty
	}
	
	lastNode := l.tail.prev
	key := lastNode.key
	
	l.removeNode(lastNode)
	delete(l.nodes, key)
	
	return key
}

// removeNode removes a node from the list (helper function)
func (l *lruList) removeNode(node *lruNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

// size returns the number of items in the list
func (l *lruList) size() int {
	return len(l.nodes)
}