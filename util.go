package sei_sdk

func Map[T any, I any](ss []T, callback func(T) I) []I {
	if len(ss) == 0 {
		return nil
	}

	ret := make([]I, 0, len(ss))
	for i := range ss {
		ret = append(ret, callback(ss[i]))
	}

	return ret
}

func Chunk[T any](slice []T, chunkSize int) (chunks [][]T) {
	for {
		if len(slice) == 0 {
			break
		}

		if len(slice) < chunkSize {
			chunkSize = len(slice)
		}

		chunks = append(chunks, slice[0:chunkSize])
		slice = slice[chunkSize:]
	}

	return
}
