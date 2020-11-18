package lottery

import(
	"../../RSA"
	"math/big"
	"strconv"
	"fmt"
)

func Draw(seed int64, slot int64, sk RSA.SecretKey) *big.Int {
	msg := []byte("LOTTERY" + strconv.FormatInt(seed, 10) + ":" + strconv.FormatInt(slot, 10))
	hashedMsg := RSA.MakeSHA256Hex(msg)
	msgBigInt := new(big.Int).SetBytes([]byte(hashedMsg))
	return RSA.Sign(*msgBigInt, sk)
}

func VerifyDraw(draw *big.Int, seed int64, slot int64, pk RSA.PublicKey) bool {
	unSignedMsg := new(big.Int).SetBytes([]byte("LOTTERY" + strconv.FormatInt(seed, 10) + ":" + strconv.FormatInt(slot, 10)))
	hashedUnSignedMsg := RSA.MakeSHA256Hex(unSignedMsg.Bytes())
	unSignedInt := new(big.Int).SetBytes([]byte(hashedUnSignedMsg))
	return RSA.Verify(*draw, *unSignedInt, pk)
}

func HasWonLottery(draw *big.Int, pk RSA.PublicKey, seed int64, slot int64, tickets int64) bool {
	number := "5363070104665469240904805716549079022559868994804632543650074225517082394787384364214337759530303772115756764726526000471329952411356772521747779646088243000001"
	hardness, _ := new(big.Int).SetString(number, 10)

	biggest := big.NewInt(0)
	
	if VerifyDraw(draw, seed, slot, pk) {
		a := big.NewInt(tickets)
		H := new(big.Int).SetBytes([]byte(RSA.MakeSHA256Hex(draw.Bytes())))
		
		val := new(big.Int).Mul(a, H)
		
		
		if biggest.Cmp(val) == -1 {
			biggest = val
		}


		//fmt.Println("BIGGEST:", biggest)

		// if (val > hardness) = +1 
		// if (val == hardness) = 0
		// if (val < hardness) = -1
		compare := val.Cmp(hardness)
		if compare == 1 {
			fmt.Println(val)
		}
		return compare == 1
	} else {
		return false
	}
}