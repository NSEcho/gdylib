package macho

func strippedNull(bts []byte) []byte {
	for i := 0; i < len(bts); i++ {
		if bts[i] == 0 {
			return bts[:i]
		}
	}
	return bts
}

func ZeroSlice(size int) []byte {
	s := make([]byte, size)
	for i := 0; i < size; i++ {
		s[i] = 0
	}
	return s
}
