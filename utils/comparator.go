package utils

type Comparator func(a, b interface{}) int
type Combined_key struct {
	Id    uint64
	Price float64
}

func Comparator_buy(a, b interface{}) int {
	a1 := a.(Combined_key)
	b1 := b.(Combined_key)
	if a1.Price < b1.Price {
		return 1
	} else if a1.Price == b1.Price {
		if a1.Id < b1.Id {
			return 1
		}
	}
	return -1
}

func Comparator_sell(a, b interface{}) int {
	a1 := a.(Combined_key)
	b1 := b.(Combined_key)
	if a1.Price > b1.Price {
		return 1
	} else if a1.Price == b1.Price {
		if a1.Id < b1.Id {
			return 1
		}
	}
	return -1
}
