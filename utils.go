package gomigrate

// This type is used to sort migration ids.

type uint64slice []uint64

func (u uint64slice) Len() int {
	return len(u)
}

func (u uint64slice) Less(a, b int) bool {
	return u[a] < u[b]
}

func (u uint64slice) Swap(a, b int) {
	tempA := u[a]
	u[a] = u[b]
	u[b] = tempA
}
