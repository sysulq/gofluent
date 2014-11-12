package ik

type StringVector []string

func (sv *StringVector) Append(v string) {
	oldLength := len(*sv)
	sv.ensureCapacity(oldLength + 1)
	(*sv)[oldLength] = v
}

func (sv *StringVector) Push(v string) {
	sv.Append(v)
}

func (sv *StringVector) Pop() string {
	retval := (*sv)[len(*sv) - 1]
	*sv = (*sv)[0:len(*sv) - 1]
	return retval
}

func (sv *StringVector) Shift() string {
	retval := (*sv)[0]
	*sv = (*sv)[1:len(*sv)]
	return retval
}

func (sv *StringVector) Last() string {
	return (*sv)[len(*sv) - 1]
}

func (sv *StringVector) First() string {
	return (*sv)[0]
}

func (sv *StringVector) ensureCapacity(l int) {
	if l < 256 {
		if l > cap(*sv) {
			newSlice := make([]string, l)
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
		newSlice := make([]string, newCapacity)
		copy(newSlice, *sv)
		*sv = newSlice
	}
	*sv = (*sv)[0:l]
}
