package wallet

type Set map[interface{}]bool

func NewSet() Set {
	return make(Set)
}

// Add Add the specified element to this set if it is not already present (optional operation)
func (s *Set) Add(i interface{}) bool {
	_, found := (*s)[i]
	if found {
		return false //False if it existed already
	}

	(*s)[i] = true
	return true
}

// Contains Returns true if this set contains the specified elements
func (s *Set) Contains(i ...interface{}) bool {
	for _, val := range i {
		if _, ok := (*s)[val]; !ok {
			return false
		}
	}
	return true
}
