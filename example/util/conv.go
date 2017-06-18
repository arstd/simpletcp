package util

func Uint32(bs [4]byte) uint32 {
	return uint32(bs[0])<<24 +
		uint32(bs[1])<<16 +
		uint32(bs[2])<<8 +
		uint32(bs[3])
}
func Bytes(num uint32) [4]byte {
	return [4]byte{
		byte(num >> 24),
		byte(num >> 16),
		byte(num >> 8),
		byte(num),
	}
}
