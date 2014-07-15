package main

import (
	"bytes"
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethutil"
	"os"
	"time"
)

var (
	JeffCoinAddr = ethutil.Hex2Bytes("2b984e578b430612e38d19014cff841ba39cdb9a")
	coinlogger   = ethlog.NewLogger("JEFF")
)

func exit(format string, v ...interface{}) {
	fmt.Printf(format, v...)
	os.Exit(0)
}

func mineJeffCoin(diff int, prevHash []byte, quit chan bool) (s uint64) {
	cmp := make([]byte, diff)

out:
	for ; ; s++ {
		select {
		case <-quit:
			break out
		default:
			l := ethutil.BinaryLength(int(s))
			n := ethutil.NumberToBytes(s, l*8)
			b := ethutil.LeftPadBytes(n, 32)

			h := ethcrypto.Sha3Bin(append(prevHash, b...))
			if bytes.Compare(h[:diff], cmp) == 0 {
				break out
			}
		}
	}

	return
}

func getSeed(eth *eth.Ethereum) int {
	stateObject := eth.StateManager().CurrentState().GetStateObject(JeffCoinAddr)
	if stateObject != nil {
		return int(stateObject.GetStorage(ethutil.Big("0")).Uint())
	}

	coinlogger.Warnln("JeffCoin not found on the network")

	return 0
}

func getDiff(eth *eth.Ethereum) int {
	stateObject := eth.StateManager().CurrentState().GetStateObject(JeffCoinAddr)
	if stateObject != nil {
		return int(stateObject.GetStorage(ethutil.Big("1")).Uint())
	}

	coinlogger.Warnln("JeffCoin not found on the network")

	return 0
}

func mine(ethereum *eth.Ethereum) {
	quit := make(chan bool, 1)
	block := make(chan ethutil.React, 1)
	reactor := ethereum.Reactor()
	reactor.Subscribe("newBlock", block)

	// TODO get T number from the contract
	for T := getSeed(ethereum); ; T++ {
		select {
		case <-block:
			quit <- true
		default:
			if T > 0 {
				diff := getDiff(ethereum)
				if diff > 0 {
					l := ethutil.BinaryLength(T)
					n := ethutil.NumberToBytes(int32(T), l*8)
					b := ethutil.LeftPadBytes(n, 32)
					nonce := mineJeffCoin(diff, append(ethereum.BlockChain().CurrentBlock.PrevHash, b...), quit)

					coinlogger.Debugf("%d, seed = %x\n", T, nonce)
				} else {
					time.Sleep(500 * time.Millisecond)
				}
			} else {
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
}
