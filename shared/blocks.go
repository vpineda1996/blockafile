package shared


type Block struct {
	version int
	prevBlock string
	records [][512]byte
	minerId string
	nonce string
}