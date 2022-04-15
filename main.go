package main

import (
	"time"
	bx "trader/bexchange"
)

func main() {
	actions := make(chan *bx.Action)
	done := make(chan bool)

	go bx.ConsoleActionHandler(actions, done)

	ob := bx.NewOrderBook(actions)
	//ob.AddOrder(bx.NewOrder(1, false, 50, 50))
	//ob.AddOrder(bx.NewOrder(2, false, 45, 25))
	//ob.AddOrder(bx.NewOrder(3, false, 45, 25))
	//ob.AddOrder(bx.NewOrder(4, true, 55, 75))
	//ob.AddOrder(bx.NewOrder(5, false, 50, 50))
	//ob.CancelOrder(1, 50, false)
	go func() {
		for {
			time.Sleep(30 * time.Second)
			ob.BuyPrice5()
			ob.SellPrice5()
			ob.GetFinalPrice()
		}
	}()
	ob.AddOrder(bx.NewOrder(1, true, 2082.34, 1))
	ob.AddOrder(bx.NewOrder(2, true, 2087.6, 2))
	ob.AddOrder(bx.NewOrder(3, true, 2087.8, 1))
	ob.AddOrder(bx.NewOrder(4, true, 2085.01, 5))
	ob.AddOrder(bx.NewOrder(5, true, 2088.02, 3))

	ob.AddOrder(bx.NewOrder(6, false, 2087.60, 6))
	ob.AddOrder(bx.NewOrder(7, true, 2081.77, 7))
	ob.AddOrder(bx.NewOrder(8, true, 2086.0, 3))
	ob.AddOrder(bx.NewOrder(9, true, 2088.33, 1))
	ob.AddOrder(bx.NewOrder(10, false, 2086.54, 2))
	ob.AddOrder(bx.NewOrder(11, false, 2086.55, 5))
	ob.AddOrder(bx.NewOrder(12, true, 2086.55, 3))
	ob.Done()

	<-done
}
