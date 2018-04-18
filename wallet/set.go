package wallet

type AddressSet map[string]bool

func NewAddressSet() AddressSet {
	return make(AddressSet)
}

// Add Add the specified element to this set if it is not already present (optional operation)
func (s *AddressSet) Add(i string) bool {
	_, found := (*s)[i]
	if found {
		return false //False if it existed already
	}

	(*s)[i] = true
	return true
}

// Contains Returns true if this set contains the specified elements
func (s *AddressSet) Contains(i ...string) bool {
	for _, val := range i {
		if _, ok := (*s)[val]; !ok {
			return false
		}
	}
	return true
}
