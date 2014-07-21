package main

import (
	"bytes"
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethutil"
	"os"
	"time"
)

var (
	JeffCoinAddr = ethutil.Hex2Bytes("7ae9aba14c89d5ada010922482348ea0e283bc36")
	coinlogger   = ethlog.NewLogger("JEFF")
)

type JeffCoin struct {
	state *ethchain.State
	eth   *eth.Ethereum
	fake  *ethchain.StateObject
	pub   *ethpub.PEthereum
	key   *ethcrypto.KeyPair
}

func New(ethereum *eth.Ethereum, keyPair *ethcrypto.KeyPair) *JeffCoin {
	state := ethereum.StateManager().CurrentState().Copy()
	fake := state.GetOrNewStateObject(keyPair.Address())
	fake.SetGasPool(ethutil.Big("100000000000000000000000000"))

	return &JeffCoin{
		eth:   ethereum,
		state: state,
		fake:  fake,
		pub:   ethpub.New(ethereum),
		key:   keyPair,
	}
}

func (self *JeffCoin) getSeed() int {
	stateObject := self.state.GetStateObject(JeffCoinAddr)

	if stateObject != nil {
		return int(stateObject.GetStorage(ethutil.Big("3")).Uint())
	}

	coinlogger.Warnln("JeffCoin not found on the network")

	return 0
}

func (self *JeffCoin) getDiff() int {
	stateObject := self.state.GetStateObject(JeffCoinAddr)
	if stateObject != nil {
		return int(stateObject.GetStorage(ethutil.Big("1")).Uint())
	}

	coinlogger.Warnln("JeffCoin not found on the network")

	return 0
}

func (self *JeffCoin) createTx(nonce []byte) (err error) {
	_, err = self.pub.Transact(ethutil.Bytes2Hex(self.key.PrivateKey), ethutil.Bytes2Hex(JeffCoinAddr), "0", "6000", "10000000000000", string(nonce))

	return
}

func (self *JeffCoin) Mine() {
	var (
		quitChan  = make(chan bool, 1)
		blockChan = make(chan ethutil.React, 1)
		txChan    = make(chan ethutil.React, 1)
		reactor   = self.eth.Reactor()
		block     = self.eth.BlockChain().CurrentBlock
		parent    = self.eth.BlockChain().GetBlock(block.PrevHash)
	)

	reactor.Subscribe("newBlock", blockChan)
	reactor.Subscribe("newTx:pre", txChan)

	for {
		select {
		case <-blockChan:
			quitChan <- true
			// Get the new Ethereum state
			self.state = self.eth.StateManager().CurrentState().Copy()
		case msg := <-txChan:
			tx := []*ethchain.Transaction{msg.Resource.(*ethchain.Transaction)}

			if bytes.Compare(tx[0].Recipient, JeffCoinAddr) == 0 {
				_, _, _, err := self.eth.StateManager().ProcessTransactions(self.fake, self.state, block, parent, tx)
				if err != nil {
					coinlogger.Warnln("Error proc tx ", err)
				}

				// A block has been found and thus the seed has probably changed
				quitChan <- true
			}
		default:
			seed := self.getSeed()
			diff := self.getDiff()
			coinlogger.Debugln("mining with diff = ", diff, " seed = ", seed)
			if diff > 0 {
				l := ethutil.BinaryLength(seed)
				n := ethutil.NumberToBytes(int32(seed), l*8)
				b := ethutil.LeftPadBytes(n, 32)
				nonce := mineJeffCoin(diff, append(self.eth.BlockChain().CurrentBlock.PrevHash, b...), quitChan)
				if len(nonce) == 32 {
					err := self.createTx(nonce)
					if err != nil {
						coinlogger.Debugln(err)
					} else {
						coinlogger.Debugf("seed = %d, diff = %d, nonce = %x\n", seed, diff, nonce)

						time.Sleep(500 * time.Millisecond)
					}
				} else {
					coinlogger.Debugln("invalid nonce len")
				}
			} else {
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
}

func mineJeffCoin(diff int, prevHash []byte, quit chan bool) (nonce []byte) {
	cmp := make([]byte, diff)

out:
	for s := uint64(0); ; s++ {
		select {
		case <-quit:
			break out
		default:
			l := ethutil.BinaryLength(int(s))
			n := ethutil.NumberToBytes(s, l*8)
			nonce = ethutil.LeftPadBytes(n, 32)

			h := ethcrypto.Sha3Bin(append(prevHash, nonce...))
			if bytes.Compare(h[:diff], cmp) == 0 {
				break out
			}
		}
	}

	return
}

func exit(format string, v ...interface{}) {
	fmt.Printf(format, v...)
	os.Exit(0)
}
