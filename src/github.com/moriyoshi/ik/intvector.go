package ik

type IntVector []int

func (sv *IntVector) Append(v int) {
	oldLength := len(*sv)
	sv.ensureCapacity(oldLength + 1)
	(*sv)[oldLength] = v
}

func (sv *IntVector) Push(v int) {
	sv.Append(v)
}

func (sv *IntVector) Pop() int {
	retval := (*sv)[len(*sv) - 1]
	*sv = (*sv)[0:len(*sv) - 1]
	return retval
}

func (sv *IntVector) Shift() int {
	retval := (*sv)[0]
	*sv = (*sv)[1:len(*sv)]
	return retval
}

func (sv *IntVector) Last() int {
	return (*sv)[len(*sv) - 1]
}

func (sv *IntVector) First() int {
	return (*sv)[0]
}

func (sv *IntVector) ensureCapacity(l int) {
	if l < 256 {
		if l > cap(*sv) {
			newSlice := make([]int, l)
			copy(newSlice, *sv)
			*sv = newSlice
		}
	} else {
		newCapacity := cap(*sv)
		if newCapacity < 256 {
			newCapacity = 128
		}
		for l > newCapacity {
			newCapacity = 2 * newCapacity
			if newCapacity < cap(*sv) {
				/* unlikely */
				panic("out of memory")
			}
		}
		newSlice := make([]int, newCapacity)
		copy(newSlice, *sv)
		*sv = newSlice
	}
	*sv = (*sv)[0:l]
}
