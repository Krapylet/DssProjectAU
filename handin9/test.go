package main

import (
	"./peer2peer/lottery"
	"./RSA"
	"fmt"
)


func main() {
	for i := 0; i < 10; i++ {
		test()
	}
}

func test() {

	seed := 1
	slot := 0
	pk, sk := RSA.KeyGen(2048)
	var tickets int64 = 1000000	
	winCounter := 0
	for i := 0; i < 10; i++ {
		draw := lottery.Draw(seed, slot, sk)
		hasWon := lottery.HasWonLottery(draw, pk, seed, slot, tickets)
		if (hasWon){ 
			winCounter++ 
			fmt.Println("Won on slot:", slot)
		}
		slot++
	}
	fmt.Println("wins:", winCounter)

}