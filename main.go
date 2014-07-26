package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path"
	"runtime"

	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
)

func DataPath() string {
	usr, _ := user.Current()
	return path.Join(usr.HomeDir, ".ethereum")
}

var (
	Datadir = DataPath()
)

const (
	ClientIdentifier = "JeffCoin NC"
	Version          = "0.1"
	ConfigFile       = "conf.ini"
	Identifier       = "Jeff"
	LogLevel         = 4
	OutboundPort     = "40404"
	MaxPeers         = 10
	UseSeed          = true
)

var logger = ethlog.NewLogger("CLI")
var keyMarager *ethcrypto.KeyManager
var interruptCallbacks = []func(os.Signal){}

func handleInterrupt() {
	c := make(chan os.Signal, 1)
	go func() {
		signal.Notify(c, os.Interrupt)
		for sig := range c {
			logger.Errorf("Shutting down (%v) ... \n", sig)
			for _, cb := range interruptCallbacks {
				cb(sig)
			}
		}
	}()
}

func registerInterrupt(cb func(os.Signal)) {
	interruptCallbacks = append(interruptCallbacks, cb)
}

func initDataDir(Datadir string) {
	_, err := os.Stat(Datadir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Data directory '%s' doesn't Exitst, creating it\n", Datadir)
			os.Mkdir(Datadir, 0777)
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	handleInterrupt()

	initDataDir(Datadir)

	ethutil.ReadConfig(ConfigFile, Datadir, "ETH")

	ethlog.AddLogSystem(ethlog.NewStdLogSystem(os.Stdout, log.LstdFlags, ethlog.LogLevel(LogLevel)))

	db, err := ethdb.NewLDBDatabase("database")
	if err != nil {
		logger.Fatalln(err)
	}

	keyManager := ethcrypto.NewDBKeyManager(db)
	err = keyManager.Init("", 0, false)
	if err != nil {
		panic(err)
	}
	fmt.Printf("addr %x\n", keyManager.KeyPair().Address())

	clientIdentity := ethwire.NewSimpleClientIdentity(ClientIdentifier, Version, Identifier)

	ethereum, err := eth.New(db, clientIdentity, keyManager, eth.CapDefault, false)
	if err != nil {
		logger.Fatalln("eth start err:", err)
	}
	ethereum.Port = OutboundPort
	ethereum.MaxPeers = MaxPeers

	logger.Infof("Starting %s", ethereum.ClientIdentity())
	ethereum.Start(UseSeed)
	registerInterrupt(func(sig os.Signal) {
		ethereum.Stop()
		ethlog.Flush()
	})

	ethereum.ConnectToPeer("localhost:30303")

	jeffcoin := New(ethereum, keyManager.KeyPair())
	jeffcoin.Start()

	// this blocks the thread
	ethereum.WaitForShutdown()
	ethlog.Flush()
}
