package main

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethpipe"
	"github.com/ethereum/eth-go/ethreact"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethtrie"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethvm"
)

var (
	JeffCoinAddr []byte
	coinlogger   = ethlog.NewLogger("JEFF")
)

func Val(v interface{}) *ethutil.Value {
	return ethutil.NewValue(v)
}

type JeffCoin struct {
	state *ethstate.State
	eth   *eth.Ethereum
	pipe  *ethpipe.Pipe
	key   *ethcrypto.KeyPair

	mineStopChan chan bool
	IsMining     bool
}

func New(ethereum *eth.Ethereum, keyPair *ethcrypto.KeyPair) *JeffCoin {
	var (
		state = ethstate.New(ethtrie.New(ethutil.Config.Db, ethereum.StateManager().CurrentState().Root()))
		pipe  = ethpipe.New(ethereum)
	)
	pipe.Vm.State = state

	return &JeffCoin{
		eth:          ethereum,
		state:        state,
		pipe:         pipe,
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

		JeffCoinAddr, err := self.pipe.Transact(self.key, nil, Val(0), Val(6000), Val(10000000000000), []byte(code))
		if err != nil {
			exit("initial %v\n", err)
		}

		coinlogger.Infof("init contract %x\n", JeffCoinAddr)
		err = ethutil.WriteFile(addrPath, []byte(ethutil.Bytes2Hex(JeffCoinAddr)))
		if err != nil {
			exit("err write %v\n", err)
		}
	}

	pipe := self.pipe
	cfg := pipe.World().Config()
	fmt.Printf("NameReg, JeffAddr = %x\n", cfg.Get("NameReg").StorageString("Jeff").Bytes())
}

func (self *JeffCoin) Balance() int32 {
	object := self.pipe.World().Get(JeffCoinAddr)
	if object != nil {
		addr := Val(1000).Add(self.key.Address())
		value := object.StorageValue(addr)

		return int32(value.Uint())
	}

	return 0
}

func (self *JeffCoin) getSeed() int {
	object := self.pipe.World().Get(JeffCoinAddr)
	if object != nil {
		return int(object.StorageValue(Val(3)).Uint())
	}

	exit("err not found")

	return 0
}

func (self *JeffCoin) getDiff() int {
	object := self.pipe.World().Get(JeffCoinAddr)
	if object != nil {
		return int(object.StorageValue(Val(1)).Uint())
	}

	exit("err not found")

	return 0
}

func (self *JeffCoin) createTx(nonce []byte) (err error) {
	data := ethutil.ParseData("mine", nonce)
	_, err = self.pipe.Transact(self.key, JeffCoinAddr, Val(0), Val(6000), Val(10000000000000), data)

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
		blockChan = make(chan ethreact.Event, 1)
		txChan    = make(chan ethreact.Event, 1)
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
			self.state = ethstate.New(ethtrie.New(ethutil.Config.Db, self.eth.StateManager().CurrentState().Root()))
			block = msg.Resource.(*ethchain.Block)
		case msg := <-txChan:
			tx := msg.Resource.(*ethchain.Transaction)

			if bytes.Compare(tx.Recipient, JeffCoinAddr) == 0 {
				object := self.pipe.World().Get(JeffCoinAddr)
				_, err := self.pipe.ExecuteObject(object, tx.Data, Val(0), Val(1000000), Val(0))
				if err != nil {
					coinlogger.Infoln(err)
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
