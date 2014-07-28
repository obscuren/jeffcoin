package main

import (
	"fmt"
	"time"

	"github.com/ethereum/eth-go"
	"github.com/go-qml/qml"
)

func UiInit() {
	qml.Init(nil)
}

type Container struct {
	win       *qml.Window
	engine    *qml.Engine
	context   *qml.Context
	component *qml.Common

	eth *eth.Ethereum

	jc *JeffCoin
}

func NewContainer(eth *eth.Ethereum, jeffCredit *JeffCoin) *Container {
	engine := qml.NewEngine()
	context := engine.Context()

	context.SetVar("jc", jeffCredit)

	return &Container{
		eth: eth,

		engine:  engine,
		context: context,
		jc:      jeffCredit,
	}
}

func (self *Container) Show() error {
	comp, err := self.engine.LoadFile("layout/main.qml")
	if err != nil {
		fmt.Println(err)
		return err
	}

	self.win = comp.CreateWindow(nil)
	self.win.Show()

	go self.update()

	self.win.Wait()

	return nil
}

func (self *Container) setBalanceLabel() {
	self.object("balanceLabel").Set("text", fmt.Sprintf("JΞF: %d", self.jc.Balance()))
}

func (self *Container) object(name string) qml.Object {
	return self.win.Root().ObjectByName(name)
}

func (self *Container) update() {
	for {
		select {
		default:
			self.setBalanceLabel()

			time.Sleep(500 * time.Millisecond)
		}
	}
}
