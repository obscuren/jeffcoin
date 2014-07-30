package main

import (
	"bytes"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethtrie"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethvm"
)

var (
	JeffCoinAddr []byte
	coinlogger   = ethlog.NewLogger("JEFF")
)

type JeffCoin struct {
	state *ethstate.State
	eth   *eth.Ethereum
	fake  *ethstate.StateObject
	pub   *ethpub.PEthereum
	key   *ethcrypto.KeyPair

	mineStopChan chan bool
	IsMining     bool
}

func New(ethereum *eth.Ethereum, keyPair *ethcrypto.KeyPair) *JeffCoin {
	state := ethstate.NewState(ethtrie.NewTrie(ethutil.Config.Db, ethereum.StateManager().CurrentState().Root()))
	fake := state.GetOrNewStateObject(keyPair.Address())

	return &JeffCoin{
		eth:          ethereum,
		state:        state,
		fake:         fake,
		pub:          ethpub.New(ethereum),
		key:          keyPair,
		mineStopChan: make(chan bool, 1),
	}
}

func (self *JeffCoin) WaitForCatchUp() {
	time.Sleep(2 * time.Second)

	for !self.eth.IsUpToDate() {
		time.Sleep(500 * time.Millisecond)
	}
}

func (self *JeffCoin) Start() {
	addrPath := ethutil.ExpandHomePath("~/.jeffcoin/addr")
	if ethutil.FileExist(addrPath) {
		addr, _ := ethutil.ReadAllFile(addrPath)
		JeffCoinAddr = ethutil.Hex2Bytes(addr[0 : len(addr)-1])

		coinlogger.Infof("Found addr file %x\n", JeffCoinAddr)

		self.WaitForCatchUp()

		coinlogger.Infoln("Ethereum is up to date")
	} else {
		self.WaitForCatchUp()

		coinlogger.Infoln("Ethereum is up to date")

		code, err := ethutil.ReadAllFile("./contract.mu")
		if err != nil {
			exit("cannot read contract.mu %v\n", err)
		}

		receipt, err := self.pub.Create(ethutil.Bytes2Hex(self.key.PrivateKey), "0", "6000", "10000000000000", code)
		if err != nil {
			exit("initial %v\n", err)
		}

		JeffCoinAddr = ethutil.Hex2Bytes(receipt.Address)
		coinlogger.Infof("init contract %x\n", JeffCoinAddr)
		err = ethutil.WriteFile(addrPath, []byte(ethutil.Bytes2Hex(JeffCoinAddr)))
		if err != nil {
			exit("err write %v\n", err)
		}
	}
}

func (self *JeffCoin) Balance() int32 {
	stateObject := self.state.GetStateObject(JeffCoinAddr)
	if stateObject != nil {
		addr := ethutil.NewValue(1000).Add(self.fake.Address())
		value := stateObject.GetStorage(addr.BigInt())

		return int32(value.Uint())
	}

	return 0
}

func (self *JeffCoin) getSeed() int {
	stateObject := self.state.GetStateObject(JeffCoinAddr)

	if stateObject != nil {
		return int(stateObject.GetStorage(ethutil.Big("3")).Uint())
	}

	exit("err not found")

	return 0
}

func (self *JeffCoin) getDiff() int {
	stateObject := self.state.GetStateObject(JeffCoinAddr)
	if stateObject != nil {
		return int(stateObject.GetStorage(ethutil.Big("1")).Uint())
	}

	exit("err not found")

	return 0
}

func (self *JeffCoin) createTx(nonce []byte) (err error) {
	data := ethutil.ParseData("mine", nonce)
	_, err = self.pub.Transact(ethutil.Bytes2Hex(self.key.PrivateKey), ethutil.Bytes2Hex(JeffCoinAddr), "0", "6000", "10000000000000", "0x"+ethutil.Bytes2Hex(data))

	return
}

func (self *JeffCoin) StartMiner() {
	self.IsMining = true
	go self.Mine()
}

func (self *JeffCoin) StopMiner() {
	self.IsMining = false
	self.mineStopChan <- true
}

func (self *JeffCoin) Mine() {
	var (
		quitChan  = make(chan bool, 1)
		blockChan = make(chan ethutil.React, 1)
		txChan    = make(chan ethutil.React, 1)
		reactor   = self.eth.Reactor()
		block     = self.eth.BlockChain().CurrentBlock

		env = NewEnv(self.state, block)
		vm  = ethvm.New(env)
	)
	vm.Verbose = true

	reactor.Subscribe("newBlock", blockChan)
	reactor.Subscribe("newTx:pre", txChan)

out:
	for {
		select {
		case <-self.mineStopChan:
			quitChan <- true
			break out
		case msg := <-blockChan:
			quitChan <- true
			// Get the new Ethereum state
			self.state = ethstate.NewState(ethtrie.NewTrie(ethutil.Config.Db, self.eth.StateManager().CurrentState().Root()))
			block = msg.Resource.(*ethchain.Block)
		case msg := <-txChan:
			tx := msg.Resource.(*ethchain.Transaction)

			if bytes.Compare(tx.Recipient, JeffCoinAddr) == 0 {
				object := self.state.GetStateObject(JeffCoinAddr)
				callerClosure := ethvm.NewClosure(self.fake, object, object.Code, big.NewInt(1000000), big.NewInt(0))

				_, _, e := callerClosure.Call(vm, tx.Data)
				if e != nil {
					fmt.Println("error", e)
				}

				// A block has been found and thus the seed has probably changed
				quitChan <- true
			}
		default:
			seed := self.getSeed()
			diff := self.getDiff()
			coinlogger.Debugln("mining with diff = ", diff, " seed = ", seed)
			if diff > 0 {
				n := ethutil.NumberToBytes(int64(seed), 64)
				b := ethutil.LeftPadBytes(n, 32)
				nonce := mineJeffCoin(diff, b, quitChan)
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

func mineJeffCoin(diff int, seed []byte, quit chan bool) (nonce []byte) {
	cmp := make([]byte, diff)

out:
	for s := uint64(0); ; s++ {
		select {
		case <-quit:
			break out
		default:
			n := ethutil.NumberToBytes(s, 64)
			nonce = ethutil.LeftPadBytes(n, 32)

			h := ethcrypto.Sha3Bin(append(nonce, seed...))
			if bytes.Compare(h[:diff], cmp) == 0 {
				fmt.Printf("SHA3( %x )\n", h)
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
