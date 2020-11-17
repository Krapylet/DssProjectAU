package lottery

import(
	"../../RSA"
	"math/big"
	"strconv"
)

func Draw(seed int, slot int, sk RSA.SecretKey) *big.Int {
	msg := []byte("LOTTERY" + strconv.Itoa(seed) + ":" + strconv.Itoa(slot))
	msgBigInt := new(big.Int).SetBytes(msg)
	return RSA.Sign(*msgBigInt, sk)
}

func VerifyDraw(draw *big.Int, seed int, slot int, pk RSA.PublicKey) bool {
	unSignedMsg := new(big.Int).SetBytes([]byte("LOTTERY" + strconv.Itoa(seed) + ":" + strconv.Itoa(slot)))
	return RSA.Verify(*draw, *unSignedMsg, pk)
}

func HasWonLottery(draw *big.Int, pk RSA.PublicKey, seed int, slot int, tickets int64) bool {
	
	hardness := big.NewInt(439734513971000000)
	if VerifyDraw(draw, seed, slot, pk) {
		a := big.NewInt(tickets)
		H := new(big.Int).SetBytes([]byte(RSA.MakeSHA256Hex(draw.Bytes())[:5]))
		val := new(big.Int).Mul(a, H)

		// if (val > hardness) = +1 
		// if (val == hardness) = 0
		// if (val < hardness) = -1
		compare := val.Cmp(hardness)
		return compare == 1
	} else {
		return false
	}
}