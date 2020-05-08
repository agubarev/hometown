package task

type Queue []Task

func (q Queue) Len() int {
	return len(q)
}

func (q Queue) Less(i, j int) bool {
	// TODO: implement sort based on operation per object priority (i.e.: read, write, delete, etc.)
	// TODO: discriminate based on object designators and read/write operations

	return true
}

func (q Queue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}
